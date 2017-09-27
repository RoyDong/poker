package lib

import (
    "bytes"
    "fmt"
    "strings"
    "net/smtp"
)

type MailConfig struct {
    Name      string
    Server    string
    Username  string
    Password  string
    Host      string
    HostName  string // 本机hostname
}

type Mail struct {
    Subject string
    Content string
    Receivers []string
    Sender    string
}

type Mailer struct {
    conf MailConfig
    mailPipe chan Mail
    inLoop bool
}

func NewMailer(conf MailConfig) *Mailer {
    mailer := &Mailer{conf: conf, mailPipe: make(chan Mail, 50)}
    mailer.StartLoop()
    return mailer
}

func (this *Mailer) Send(m Mail) {
    this.mailPipe <- m
}

func (this *Mailer) StartLoop() {
    this.inLoop = true
    go this.mailLoop()
}

func (this *Mailer) StopLoop() {
    this.inLoop = false
}

func (this *Mailer) mailLoop() {
    for this.inLoop {
        mail := <- this.mailPipe
        var data bytes.Buffer
        fmt.Fprintf(&data, "To: %s\r\n", strings.Join(mail.Receivers, ","))
        fmt.Fprintf(&data, "From: %s\r\n", mail.Sender)
        fmt.Fprintf(&data, "Subject: %s\r\n", mail.Subject)
        fmt.Fprintf(&data, "Content-Type: text/plain; charset=UTF-8\r\n")
        //fmt.Fprintf(&data, "\r\n%s", this.conf.HostName)
        fmt.Fprintf(&data, "\r\n%s", mail.Content)
        auth := smtp.PlainAuth(this.conf.Username, this.conf.Username, this.conf.Password, this.conf.Host)
        smtp.SendMail(this.conf.Server, auth, mail.Sender, mail.Receivers, data.Bytes())
    }
}
