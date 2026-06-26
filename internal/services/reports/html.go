package reports

import (
	"fmt"
	"strings"
)

func RenderDeliveryHTML(report DeliveryReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>Aivar Delivery Report</title>
<style>body{font-family:system-ui;max-width:960px;margin:2rem auto;padding:0 1rem}
table{border-collapse:collapse;width:100%%;margin:1rem 0}th,td{border:1px solid #ccc;padding:8px;text-align:left}</style></head><body>
<h1>Aivar Shield Delivery Report</h1>
<p><strong>Generated:</strong> %s</p>
<p><strong>Repo:</strong> %s</p>
<p>Approved: %d · Pending: %d · Rejected: %d · Audit events: %d</p>
<h2>Suppressions</h2>
<table><tr><th>Ref</th><th>Rule</th><th>Tool</th><th>Status</th><th>Requested by</th><th>Approved by</th><th>Reason</th></tr>`,
		report.GeneratedAt.Format("2006-01-02 15:04 UTC"),
		report.Repo,
		report.Summary.Approved, report.Summary.Pending, report.Summary.Rejected, report.Summary.AuditEvents,
	)
	for _, s := range report.Suppressions {
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			s.PlatformRef, s.RuleID, s.Tool, s.Status, s.RequestedBy, s.ApprovedBy, s.Reason)
	}
	b.WriteString(`</table><h2>Recent audit</h2><table><tr><th>Time</th><th>Actor</th><th>Action</th><th>Repo</th></tr>`)
	for _, a := range report.Audit {
		fmt.Fprintf(&b, `<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`,
			a.Timestamp.Format("2006-01-02 15:04"), a.Actor, a.Action, a.Repo)
	}
	b.WriteString(`</table></body></html>`)
	return b.String()
}
