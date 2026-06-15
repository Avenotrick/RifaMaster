package main

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
)

func sendConfirmationEmail(to, name string, number int) {
	if !cfg.SmtpEnabled() {
		log.Printf("[EMAIL SIMULADO] Para %s (%s): Número #%d confirmado", name, to, number)
		return
	}

	subject := fmt.Sprintf("Confirmación: Tu número %d para la Rifa", number)
	body := buildEmailHTML(name, number)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		cfg.SmtpFrom, to, subject, body)

	auth := smtp.PlainAuth("", cfg.SmtpUser, cfg.SmtpPass, cfg.SmtpHost)
	addr := fmt.Sprintf("%s:%d", cfg.SmtpHost, cfg.SmtpPort)

	if err := smtp.SendMail(addr, auth, cfg.SmtpFrom, []string{to}, []byte(msg)); err != nil {
		log.Printf("Error enviando email a %s: %v", to, err)
		return
	}

	log.Printf("Email enviado a %s (%s) para número #%d", name, to, number)
}

func buildEmailHTML(name string, number int) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:24px 0">
<tr><td align="center">
<table role="presentation" width="480" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:16px;overflow:hidden;box-shadow:0 1px 3px rgba(0,0,0,0.08)">
<tr><td style="background:linear-gradient(135deg,#4f46e5,#7c3aed);padding:32px 24px;text-align:center">
<h1 style="color:#ffffff;font-size:20px;margin:0;letter-spacing:-0.3px">RifaMaster</h1>
<p style="color:#c4b5fd;font-size:12px;margin:4px 0 0">Confirmación de participación</p>
</td></tr>
<tr><td style="padding:32px 24px;text-align:center">
<p style="color:#1e293b;font-size:14px;margin:0 0 8px">Hola <strong style="color:#4f46e5">%s</strong>,</p>
<p style="color:#475569;font-size:13px;margin:0 0 20px">¡Tu pago fue recibido con éxito!</p>
<div style="background:#eef2ff;border:2px solid #c7d2fe;border-radius:16px;padding:20px;margin:0 0 24px;display:inline-block">
<p style="color:#4338ca;font-size:10px;margin:0 0 2px;text-transform:uppercase;letter-spacing:1px;font-weight:600">Tu número</p>
<p style="color:#4f46e5;font-size:56px;margin:0;font-weight:900;letter-spacing:-2px;line-height:1">%02d</p>
</div>
<table role="presentation" cellpadding="0" cellspacing="0" style="text-align:left;width:100%%;background:#f8fafc;border-radius:12px;padding:16px">
<tr><td style="padding:6px 0">
<span style="color:#94a3b8;font-size:11px;text-transform:uppercase;letter-spacing:0.5px">Estado</span><br>
<span style="color:#16a34a;font-size:13px;font-weight:600">Confirmado</span>
</td></tr>
<tr><td style="padding:6px 0">
<span style="color:#94a3b8;font-size:11px;text-transform:uppercase;letter-spacing:0.5px">Rango de números</span><br>
<span style="color:#1e293b;font-size:13px;font-weight:600">1 – 1000</span>
</td></tr>
</table>
</td></tr>
<tr><td style="background:#f8fafc;padding:20px 24px;text-align:center;border-top:1px solid #e2e8f0">
<p style="color:#94a3b8;font-size:11px;margin:0;line-height:1.5">Si tenés alguna duda, contactate con el organizador de la rifa.</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`, escHTML(name), number)
}

func escHTML(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"'", "&#39;",
		"\"", "&quot;",
	)
	return r.Replace(s)
}
