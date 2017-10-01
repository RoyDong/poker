package context

type Config struct {
    Server struct {
        Hostname string
        Host     string
        SockFile string
        MaxConn  int
        Timeout  int
        Debug    bool
        PProfHost string
    }

    AlertMail struct {
        Username string
        Password string
        Subject  string
        Host     string
        Server   string
        Sender   string
        Receiver []string
    }

    Log struct {
        LogRotate string
        LogDir    string
    }

    Market struct {
        Okex struct{
            HttpHost  string
            ApiKey    string
            ApiSecret string
        }
    }
}
