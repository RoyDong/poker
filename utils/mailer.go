package utils

import (
    "bytes"
    "fmt"
    "strings"
    "net/smtp"
    "crypto/tls"
)

type MailConfig struct {
    Name      string
    Server    string
    UseSsl    bool
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
    errorPipe chan error
}

func NewMailer(conf MailConfig) *Mailer {
    mailer := &Mailer{conf: conf, mailPipe: make(chan Mail, 50)}
    mailer.errorPipe = make(chan error, 50)
    mailer.StartLoop()
    return mailer
}

func (this *Mailer) Send(m Mail) {
    this.mailPipe <- m
}

func (this *Mailer) StartLoop() {
    if this.inLoop {
        return
    }
    this.inLoop = true
    go this.mailLoop()
}

func (this *Mailer) StopLoop() {
    this.inLoop = false
}

func (this *Mailer) mailLoop() {
    for this.inLoop {
        mail := <- this.mailPipe
        var err error
        if this.conf.UseSsl {
            err = this.sendSsl(mail)
        } else {
            err = this.send(mail)
        }
        if err != nil && len(this.errorPipe) < cap(this.errorPipe) {
            this.errorPipe <-err
        }
    }
}

func (this *Mailer) send(m Mail) error {
    auth := smtp.PlainAuth(this.conf.Username, this.conf.Username, this.conf.Password, this.conf.Host)
    return smtp.SendMail(this.conf.Server, auth, m.Sender, m.Receivers, this.packMail(m))
}

func (this *Mailer) packMail(m Mail) []byte {
    var data bytes.Buffer
    fmt.Fprintf(&data, "To: %s\r\n", strings.Join(m.Receivers, ","))
    fmt.Fprintf(&data, "From: %s\r\n", m.Sender)
    fmt.Fprintf(&data, "Subject: %s\r\n", m.Subject)
    fmt.Fprintf(&data, "Content-Type: text/plain; charset=UTF-8\r\n")
    //fmt.Fprintf(&data, "\r\n%s", this.conf.HostName)
    fmt.Fprintf(&data, "\r\n%s", m.Content)
    return data.Bytes()
}

func (this *Mailer) sendSsl(m Mail) error {
    conn, err := tls.Dial("tcp", this.conf.Server, nil)
    smtpClient, err := smtp.NewClient(conn, this.conf.Host)
    if err != nil {
        return err
    }
    auth := smtp.PlainAuth(this.conf.Username, this.conf.Username, this.conf.Password, this.conf.Host)
    err = smtpClient.Auth(auth)
    if err != nil {
        return err
    }
    for _, addr := range m.Receivers {
        if err = smtpClient.Rcpt(addr); err != nil {
            return err
        }
    }
    w, err := smtpClient.Data()
    if err != nil {
        return err
    }
    _, err = w.Write(this.packMail(m))
    if err != nil {
        return err
    }

    err = w.Close()
    if err != nil {
        return err
    }
    return smtpClient.Quit()
}

func (this *Mailer) Error() error {
    return <-this.errorPipe
}



