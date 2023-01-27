package main

import (
   "os"
   "io"
   "log"
   "fmt"
   "flag"
   "time"
   "context"
   "net/http"
   "os/signal"
   "syscall"
   "github.com/gin-gonic/gin"
   "github.com/influxdata/influxdb/client/v2"
   "github.com/gin-contrib/cors"
   _ "github.com/go-sql-driver/mysql"
   "database/sql"
   _ "reflect"
)

const (
   Jandi_sess_token = "4fad5d76ba9ced473717d25c65e63e71"
   Jandi_event_token = "8608a616856085e3ccda2ecd0cd9296c"
   dbmt_token = "ccuo21f4Dhzyff78FSCIlzBGw+159UhRuLyev"
   Repo_database = "oracle"
)

type Api struct {
     db * sql.DB
}

type Jandi_Req struct {
     Token       string    `json:"token" binding:"required"`
     TN          string    `json:"teamName"`
     RN          string    `json:"rootName"`
     WN          string    `json:"writerName"`
     WE          string    `json:"writerEmail"`
     Text        string    `json:"text" binding:"required"`      
     Keyword     string    `json:"keyword" binding:"required"`
     Dt          time.Time `json:"createAt" time_format: "2006-01-02"`
}

type Jandi_content struct {
     Title       string    `json:"title"`
     Description string    `json:"description"`
}

type Jandi_Resp struct {
     Body         string   `json:"body"`
     ConnectColor string   `json:"connectColor"`
     ConnectInfo  []Jandi_content  `json:"connectInfo"`
}

type DataReq struct {
     Token       string    `form:"token" json:"token" binding:"required"`
}

type User struct {
     Token       string    `form:"token" json:"token" binding:"required"`
     Id          string    `json:"id"`
     Hostname    string    `json:"hostname"`
     Service     string    `json:"service"`
     Servdesc    string    `json:"desc"`
     Userid      string    `json:"userid"`
     Userpass    string    `json:"userpass"`
     Dbtype      string    `json:"dbtype"`
     Ctime       string    `json:"ctime" time_format: "2006-01-02"`
     Utime       string    `json:"utime" time_format: "2006-01-02"`
}

type Server struct {
     Id          string    `json:"id"`
     Hostname    string    `json:"hostname"`
     Servip      string    `json:"server_ip"`
     Ipmiip      string    `json:"ipmi_ip"`
     Hosttype    string    `json:"host_type"`
     Apptype     string    `json:"app_type"`
     Service     string    `json:"service"`
     Ctime       string    `json:"ctime" time_format: "2006-01-02"`
     Utime       string    `json:"utime" time_format: "2006-01-02"`
}

func SetEnv(v_repodb_url, v_db_user, v_db_pass string) gin.HandlerFunc {
   return func(c *gin.Context) {
      o_time := time.Now()

      c.Set("repodb_url", v_repodb_url)
      c.Set("db_user", v_db_user)
      c.Set("db_pass", v_db_pass)

      c.Next()

      o_latency := time.Since(o_time)
      log.Printf(" | %d | %10s | %15s | %15s | %s %s", c.Writer.Status(), 
        o_latency, c.ClientIP(), c.ContentType(), c.Request.Method, c.Request.URL.Path)
   }
}

func usage() {
   fmt.Println("./extchan -repodb [influx-ip:port] -datadb [mysql-ip:port] -user [username] -pass [password] --log [logpath]")
}

func main() {

   var s_Logfile string 

   p_repodb := flag.String("repodb", "localhost:8086", "influx db")
   p_datadb := flag.String("datadb", "localhost:3306", "mysql db")
   p_username := flag.String("user", "admin", "influx user")
   p_password := flag.String("pass", "", "influx user pass")
   p_logfile := flag.String("log", "", "log file path")
   p_help := flag.Bool("help", false, "display help")
   flag.Parse()

   if *p_help == true {
      usage()
      os.Exit(0)
   }

   if *p_logfile != "" {
      s_Logfile = *p_logfile 
   } else {
      s_Logfile = "log/extchan.log"
   } 

   gin.DisableConsoleColor()

   o_File, err := os.OpenFile(s_Logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
   if err != nil {
      fmt.Printf("Don't make log file : %s\n", s_Logfile)
      os.Exit(0)
   }

   gin.DefaultWriter = io.MultiWriter(o_File, os.Stdout)
   gin.SetMode(gin.ReleaseMode)
   log.SetOutput(gin.DefaultWriter)

   o_Router := gin.New()
   o_Router.Use(SetEnv(*p_repodb, *p_username, *p_password))
   o_Router.Use(gin.Recovery())
   o_Router.Use(cors.Default())

   s_conn_url := fmt.Sprintf("%s:%s@tcp(%s)/dbmt", *p_username, *p_password, *p_datadb)
   o_db, err := sql.Open("mysql", s_conn_url)
   if err != nil {
      LogPrint("Data db can't connect : " + s_conn_url)
      os.Exit(0)
   }
   defer o_db.Close()

   o_db.SetMaxIdleConns(5)
   o_db.SetMaxOpenConns(5)
   o_db.SetConnMaxLifetime(time.Hour)

   o_api := &Api{ db: o_db }
  
   o_Router.GET("/test", func(c *gin.Context) {
      c.JSON(200, gin.H{ "ping" : "pong", })
   })

   o_Jandi := o_Router.Group("/jandi") 
   {
      o_Jandi.POST("/oracle", Oracle_Handler)

   }

   o_dbmt := o_Router.Group("/db")
   {
      o_dbmt.POST("/users", o_api.UserList)
      o_dbmt.POST("/adduser", o_api.UserAdd)
      o_dbmt.POST("/udtuser", o_api.UserUdt)
      o_dbmt.POST("/servers", o_api.ServerList)
   }

   o_Server := &http.Server {
      Addr:          ":8081",
      Handler:       o_Router,
      ReadTimeout:   10 * time.Second,
      WriteTimeout:  10 * time.Second,
   }

   go func() {
      o_Server.ListenAndServe()
   }()
   LogPrint("Server start...")

   o_quit := make(chan os.Signal)
   signal.Notify(o_quit, syscall.SIGINT, syscall.SIGTERM)
 
   <- o_quit
   LogPrint("Shutting down server...")

   o_ctx, o_cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer o_cancel()

   if err := o_Server.Shutdown(o_ctx); err != nil {
      LogPrint("Server forced to shutdown :" + err.Error())
   }
     
   LogPrint("Server exit... bye.")
}


func (v_db *Api) UserList(c *gin.Context) {

   var o_Msg DataReq
   o_Resp := make([]User, 0)

   if err := c.ShouldBindJSON(&o_Msg); err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusBadRequest, gin.H { "err" : "can't bind parameter for token" })
      return
   }

   if o_Msg.Token != dbmt_token {
      LogPrint("Wrong token : " + o_Msg.Token)
      c.JSON(http.StatusBadRequest, gin.H { "err" : "miss match for authienticaton token" })      
      return
   }

   o_rows, err := v_db.db.Query("select * from users")
   defer o_rows.Close()

   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql" })
      return
   }

   for o_rows.Next() {
      var o_user User

      o_rows.Scan(&o_user.Id, &o_user.Hostname, &o_user.Service, &o_user.Servdesc, &o_user.Userid, &o_user.Userpass, &o_user.Dbtype, &o_user.Ctime, &o_user.Utime)

      o_Resp = append(o_Resp, o_user)
   }


   c.JSON(http.StatusOK, o_Resp)
}

func (v_db *Api) UserAdd(c *gin.Context) {
   var o_Msg User
   var s_Sql string = "insert into users(hostname, service, servdesc, userid, userpass, dbtype) values (?,?,?,?,?,?)"

   if err := c.ShouldBindJSON(&o_Msg); err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusBadRequest, gin.H { "err" : "can't bind parameter for token" })
      return
   }

   if o_Msg.Token != dbmt_token {
      LogPrint("Wrong token : " + o_Msg.Token)
      c.JSON(http.StatusBadRequest, gin.H { "err" : "miss match for authienticaton token" })
      return
   }

   o_Stmt, err := v_db.db.Prepare(s_Sql)
   defer o_Stmt.Close()

   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql" })
      return
   }

   o_Rs, err := o_Stmt.Exec(o_Msg.Hostname, o_Msg.Service, o_Msg.Servdesc, o_Msg.Userid, o_Msg.Userpass, o_Msg.Dbtype)
   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql exec" })
      return
   }  

   _, err = o_Rs.LastInsertId()
   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql resp" })
      return
   } 

   LogPrint(o_Msg.Userid + ": inserted")
   
   c.JSON(http.StatusOK, gin.H{ "message" : o_Msg.Userid + " inserted"})
}

func (v_db *Api) UserUdt(c *gin.Context) {
   var o_Msg User
   var s_Sql string = "update users set servdesc=?, userid=?, userpass=?, dbtype=?, utime=sysdate() where id=?"

   if err := c.ShouldBindJSON(&o_Msg); err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusBadRequest, gin.H { "err" : "can't bind parameter for token" })
      return
   }
   
   if o_Msg.Token != dbmt_token {
      LogPrint("Wrong token : " + o_Msg.Token)
      c.JSON(http.StatusBadRequest, gin.H { "err" : "miss match for authienticaton token" })
      return
   }

   o_Stmt, err := v_db.db.Prepare(s_Sql)
   defer o_Stmt.Close()

   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql" })
      return
   }

   o_Rs, err := o_Stmt.Exec(o_Msg.Servdesc, o_Msg.Userid, o_Msg.Userpass, o_Msg.Dbtype, o_Msg.Id)

   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql exec" })
      return
   }

   o_Row, err := o_Rs.RowsAffected()
 
   c.JSON(http.StatusOK, gin.H{ "message" : string(o_Row) + " updated" })
}

func (v_db *Api) ServerList(c *gin.Context) {

   var o_Msg DataReq
   o_Resp := make([]Server, 0)

   if err := c.ShouldBindJSON(&o_Msg); err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusBadRequest, gin.H { "err" : "can't bind parameter for token" })
      return
   }

   if o_Msg.Token != dbmt_token {
      LogPrint("Wrong token : " + o_Msg.Token)
      c.JSON(http.StatusBadRequest, gin.H { "err" : "miss match for authienticaton token" })
      return
   }

   o_rows, err := v_db.db.Query("select * from server order by id")
   defer o_rows.Close()

   if err != nil {
      LogPrint(err.Error())
      c.JSON(http.StatusInternalServerError, gin.H { "err" : "internal server error in related to sql" })
      return
   }


   for o_rows.Next() {
      var o_server Server

      o_rows.Scan(&o_server.Id, &o_server.Hostname, &o_server.Servip, &o_server.Ipmiip, &o_server.Hosttype, &o_server.Apptype, &o_server.Service, &o_server.Ctime, &o_server.Utime)

      o_Resp = append(o_Resp, o_server)
   }

   c.JSON(http.StatusOK, o_Resp)
}


func Oracle_Handler(c *gin.Context) {

   var o_Msg Jandi_Req
   o_Resp := Jandi_Resp{}
   o_Content := Jandi_content{}

   o_Resp.Body = "[Response]"
   o_Resp.ConnectColor = "#FAC11B"
   o_Resp.ConnectInfo = []Jandi_content{}
   o_Content.Title = "No Data"
   o_Content.Description = ""

   s_db_url := c.MustGet("repodb_url").(string)
   s_db_user := c.MustGet("db_user").(string)
   s_db_pass := c.MustGet("db_pass").(string)


   if err := c.ShouldBindJSON(&o_Msg); err != nil {
      LogPrint(err.Error())
      o_Content.Title = "Json Error"
      o_Content.Description = "Can't parser json data from jandi"
   }

   if ( o_Msg.Keyword == "sess" && o_Msg.Token == Jandi_sess_token ) {
      o_Conn, err := client.NewHTTPClient(client.HTTPConfig{
         Addr: "http://" + s_db_url, Username: s_db_user, Password: s_db_pass,
      })

      if err != nil {
         o_Content.Title = "Repo Access Error"
         o_Content.Description = "Can't access repo db"
      }
   
      defer o_Conn.Close()
 
      if (  err == nil ) {
         s_Query := "select * from session_active where time > now() - 1h group by host, oracle_sid order by desc limit 1"
         o_Query := client.Query {
            Command: s_Query,
            Database: Repo_database,
         }

         o_Ret, err := o_Conn.Query(o_Query)
         if err != nil {
            o_Content.Title = "Query Fail"
            o_Content.Description = "Can't get session info in repo db"
         } else {
            o_Content.Title = "Active Session Status"

            //o_Ret.Results --> Series, Messages, Err
            //o_Row --> Name, Tags, Columns, Values, Partial
            for i_index, o_Row := range o_Ret.Results[0].Series {
               //LogPrint(reflect.TypeOf(o_Row).String())
               o_Time, _ := time.Parse(time.RFC3339, 
                            o_Row.Values[0][0].(string))
               o_Time = o_Time.In(time.FixedZone("KST", 9*60*60))
               s_Host := o_Row.Tags["host"]
               s_Sess := o_Row.Values[0][1]
               o_Content.Description += fmt.Sprintf("%3d | %v | %-25s | %-10s\n",
                           i_index,
                           o_Time.Format("2006/01/02 - 15:04:05"),
                           s_Host, s_Sess)
            }
         }
      }
   } else if ( o_Msg.Keyword == "event" && o_Msg.Token == Jandi_event_token ) {
      o_Conn, err := client.NewHTTPClient(client.HTTPConfig{
         Addr: "http://" + s_db_url, Username: s_db_user, Password: s_db_pass,
      })

      if err != nil {
         o_Content.Title = "Repo Access Error"
         o_Content.Description = "Can't access repo db"
      }

      defer o_Conn.Close()


      if (  err == nil ) {
         s_Query := "select * from session_event where time > now() - 1h group by host, oracle_sid order by desc limit 1"
         o_Query := client.Query {
            Command: s_Query,
            Database: Repo_database,
         }

         o_Ret, err := o_Conn.Query(o_Query)
         if err != nil {
            o_Content.Title = "Query Fail"
            o_Content.Description = "Can't get event info in repo db"
         } else {
            o_Content.Title = "Current event status"

            //o_Ret.Results --> Series, Messages, Err
            //o_Row --> Name, Tags, Columns, Values, Partial
            for i_index, o_Row := range o_Ret.Results[0].Series {
               //LogPrint(reflect.TypeOf(o_Row).String())
               o_Time, _ := time.Parse(time.RFC3339,
                            o_Row.Values[0][0].(string))
               o_Time = o_Time.In(time.FixedZone("KST", 9*60*60))
               s_Host := o_Row.Tags["host"]
               s_Num := o_Row.Values[0][1]
               s_Event := o_Row.Values[0][2]
               o_Content.Description += fmt.Sprintf("%3d | %v | %-25s | %-40s | %-10s\n",
                           i_index,
                           o_Time.Format("2006/01/02 - 15:04:05"),
                           s_Host, s_Event, s_Num)
            }
         }
      }

   } else {
      o_Content.Title = "Authentification Error"
      o_Content.Description = "Can't certificate jandi token or request command"
   }

   o_Resp.ConnectInfo = append(o_Resp.ConnectInfo, o_Content)
   c.JSON(http.StatusOK, o_Resp)
}



func LogPrint(v_msg string) {
   log.Printf(" | %s", v_msg)
}
