package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aivar-shield/backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const cookieName = "aivar_session"

type Service struct {
	pool   *pgxpool.Pool
	cfg    config.Config
	states map[string]time.Time
}

func NewService(pool *pgxpool.Pool, cfg config.Config) *Service {
	return &Service{pool: pool, cfg: cfg, states: map[string]time.Time{}}
}

type Claims struct {
	UserID string `json:"uid"`
	Login  string `json:"login"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

type SessionUser struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type Membership struct {
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
}

type MeResponse struct {
	User         SessionUser  `json:"user"`
	Memberships  []Membership `json:"memberships"`
	AuthEnabled  bool         `json:"auth_enabled"`
	CanApprove   bool         `json:"can_approve"`
}

func (s *Service) Enabled() bool {
	return s.cfg.AuthEnabled()
}

func (s *Service) ConfigPayload() map[string]any {
	return map[string]any{
		"enabled":    s.Enabled(),
		"login_path": "/api/v1/auth/github/login",
	}
}

func (s *Service) StartLogin(w http.ResponseWriter, r *http.Request) error {
	if !s.Enabled() {
		return fmt.Errorf("github oauth not configured")
	}
	state := randomState()
	s.states[state] = time.Now().Add(10 * time.Minute)

	q := url.Values{
		"client_id":    {s.cfg.GitHubClientID},
		"redirect_uri": {s.callbackURL()},
		"scope":        {"read:user user:email"},
		"state":        {state},
	}
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+q.Encode(), http.StatusFound)
	return nil
}

func (s *Service) HandleCallback(w http.ResponseWriter, r *http.Request) error {
	if !s.Enabled() {
		return fmt.Errorf("github oauth not configured")
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	if code == "" || state == "" || !s.consumeState(state) {
		return fmt.Errorf("invalid oauth state")
	}

	token, err := s.exchangeCode(code)
	if err != nil {
		return err
	}
	ghUser, email, err := s.fetchGitHubUser(token)
	if err != nil {
		return err
	}

	user, err := s.upsertUser(r.Context(), ghUser, email)
	if err != nil {
		return err
	}
	if err := s.ensureMemberships(r.Context(), user); err != nil {
		return err
	}

	jwtToken, err := s.signToken(user)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    jwtToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((24 * time.Hour * 7).Seconds()),
	})
	http.Redirect(w, r, s.cfg.UIURL+"/", http.StatusFound)
	return nil
}

func (s *Service) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
}

func (s *Service) UserFromRequest(r *http.Request) (*SessionUser, error) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return nil, fmt.Errorf("not authenticated")
	}
	claims, err := s.parseToken(c.Value)
	if err != nil {
		return nil, err
	}
	return &SessionUser{ID: claims.UserID, Login: claims.Login, Email: claims.Email}, nil
}

func (s *Service) Me(ctx context.Context, r *http.Request) (MeResponse, error) {
	resp := MeResponse{AuthEnabled: s.Enabled()}
	if !s.Enabled() {
		resp.CanApprove = true
		return resp, nil
	}
	user, err := s.UserFromRequest(r)
	if err != nil {
		return resp, err
	}
	resp.User = *user
	memberships, err := s.listMemberships(ctx, user.ID)
	if err != nil {
		return resp, err
	}
	resp.Memberships = memberships
	for _, m := range memberships {
		if m.Role == "admin" || m.Role == "approver" {
			resp.CanApprove = true
			break
		}
	}
	if !resp.CanApprove {
		var teamApprover bool
		_ = s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM team_memberships
				WHERE user_id = $1 AND role IN ('admin', 'approver')
			)
		`, user.ID).Scan(&teamApprover)
		resp.CanApprove = teamApprover
	}
	return resp, nil
}

func (s *Service) CanApprove(ctx context.Context, userID, orgID, teamID string) (bool, error) {
	if !s.Enabled() {
		return true, nil
	}
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM memberships WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&role)
	if err == nil && (role == "admin" || role == "approver") {
		return true, nil
	}
	if teamID == "" {
		return false, nil
	}
	err = s.pool.QueryRow(ctx, `
		SELECT role FROM team_memberships WHERE user_id = $1 AND team_id = $2
	`, userID, teamID).Scan(&role)
	if err != nil {
		return false, nil
	}
	return role == "admin" || role == "approver", nil
}

func (s *Service) IsOrgAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	if !s.Enabled() {
		return true, nil
	}
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT role FROM memberships WHERE user_id = $1 AND organization_id = $2
	`, userID, orgID).Scan(&role)
	if err != nil {
		return false, nil
	}
	return role == "admin", nil
}

// IsPlatformAdmin returns true for env-listed admins or any org admin membership.
func (s *Service) IsPlatformAdmin(ctx context.Context, user *SessionUser) bool {
	if !s.Enabled() {
		return true
	}
	for _, login := range s.cfg.AdminGitHubLogins {
		if strings.EqualFold(login, user.Login) {
			return true
		}
	}
	memberships, err := s.listMemberships(ctx, user.ID)
	if err != nil {
		return false
	}
	for _, m := range memberships {
		if m.Role == "admin" {
			return true
		}
	}
	return false
}

func (s *Service) callbackURL() string {
	return strings.TrimRight(s.cfg.APIPublicURL, "/") + "/api/v1/auth/github/callback"
}

func (s *Service) exchangeCode(code string) (string, error) {
	body := url.Values{
		"client_id":     {s.cfg.GitHubClientID},
		"client_secret": {s.cfg.GitHubClientSecret},
		"code":          {code},
		"redirect_uri":  {s.callbackURL()},
	}
	req, _ := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(body.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var payload struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.AccessToken == "" {
		return "", fmt.Errorf("oauth token error: %s", payload.Error)
	}
	return payload.AccessToken, nil
}

type ghUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (s *Service) fetchGitHubUser(token string) (ghUser, string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ghUser{}, "", err
	}
	defer res.Body.Close()
	var user ghUser
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return ghUser{}, "", err
	}
	email := user.Email
	if email == "" {
		email, _ = s.fetchPrimaryEmail(token)
	}
	return user, email, nil
}

func (s *Service) fetchPrimaryEmail(token string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", nil
}

func (s *Service) upsertUser(ctx context.Context, gh ghUser, email string) (SessionUser, error) {
	id := uuid.NewString()
	row := s.pool.QueryRow(ctx, `
		INSERT INTO users (id, github_id, login, email, name, avatar_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (github_id) DO UPDATE
		SET email = EXCLUDED.email,
		    name = EXCLUDED.name,
		    avatar_url = EXCLUDED.avatar_url,
		    login = EXCLUDED.login
		RETURNING id, login, email, name, avatar_url
	`, id, gh.ID, gh.Login, nullString(email), nullString(gh.Name), nullString(gh.AvatarURL))

	var user SessionUser
	var name, avatar *string
	var emailVal *string
	if err := row.Scan(&user.ID, &user.Login, &emailVal, &name, &avatar); err != nil {
		return SessionUser{}, err
	}
	user.Email = deref(emailVal)
	user.Name = deref(name)
	user.AvatarURL = deref(avatar)
	return user, nil
}

func (s *Service) ensureMemberships(ctx context.Context, user SessionUser) error {
	rows, err := s.pool.Query(ctx, `SELECT id FROM organizations`)
	if err != nil {
		return err
	}
	defer rows.Close()

	role := "member"
	for _, admin := range s.cfg.AdminGitHubLogins {
		if strings.EqualFold(admin, user.Login) {
			role = "admin"
			break
		}
	}
	if role == "member" {
		for _, approver := range s.cfg.ApproverGitHubLogins {
			if strings.EqualFold(approver, user.Login) {
				role = "approver"
				break
			}
		}
	}

	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return err
		}
		_, err := s.pool.Exec(ctx, `
			INSERT INTO memberships (user_id, organization_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, organization_id) DO NOTHING
		`, user.ID, orgID, role)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Service) listMemberships(ctx context.Context, userID string) ([]Membership, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT organization_id, role FROM memberships WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Membership, 0)
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.OrganizationID, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Service) signToken(user SessionUser) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Login:  user.Login,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.SessionSecret))
}

func (s *Service) parseToken(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(s.cfg.SessionSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func randomState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) consumeState(state string) bool {
	exp, ok := s.states[state]
	if !ok || time.Now().After(exp) {
		return false
	}
	delete(s.states, state)
	return true
}

func nullString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
