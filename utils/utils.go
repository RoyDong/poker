package utils

import (
    "dw/poker/context"
    "database/sql"
    "fmt"
    "net/url"
)


var NoticeLog *Logger
var AccessLog *Logger
var DebugLog *Logger
var WarningLog *Logger
var FatalLog *Logger
var SysMailer *Mailer
var sysMail Mail


var MainDB *sql.DB

func InitLog(dir, rotate, filePrefix string, debug bool) {
    DebugLog   = NewLogger(dir, filePrefix + "debug", rotate, true)
    WarningLog = NewLogger(dir, filePrefix + "warning", rotate, true)
    FatalLog   = NewLogger(dir, filePrefix + "fatal", rotate, true)
    AccessLog  = NewLogger(dir, filePrefix + "access", rotate, false)
    NoticeLog  = NewLogger(dir, filePrefix + "notice", rotate, false)
    if !debug {
        DebugLog.Mute()
    }
}

func Init(conf *context.Config) error {
    mailConf := MailConfig{}
    mailConf.Username = conf.AlertMail.Username
    mailConf.Password = conf.AlertMail.Password
    mailConf.UseSsl = conf.AlertMail.UseSsl
    mailConf.Host = conf.AlertMail.Host
    mailConf.Server = conf.AlertMail.Server
    mailConf.HostName = conf.Server.Hostname

    SysMailer = NewMailer(mailConf)
    sysMail = Mail{}
    sysMail.Sender = conf.AlertMail.Sender
    sysMail.Receivers = conf.AlertMail.Receiver
    sysMail.Subject = conf.AlertMail.Subject

    c := conf.Sqldb.Main
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s", c.Username, c.Password,
        c.Host, c.Port, c.Dbname, c.Charset)
    if len(c.Local) > 0 {
        dsn = fmt.Sprintf("%s&parseTime=true&loc=%s", dsn, url.QueryEscape(c.Local))
    }
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return err
    }
    MainDB = db
    return nil
}

//content, subject, receivers...
func SendSysMail(args ...string) {
    m := sysMail
    if len(args) > 0 {
        m.Content = []byte(args[0])
    }
    if len(args) > 1 {
        m.Subject = args[1]
    }
    if len(args) > 2 {
        m.Receivers = append(m.Receivers, args[2:]...)
    }
    SysMailer.Send(m)
}


