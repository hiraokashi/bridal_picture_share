// main.go
package main
import (
    "io" 
    "io/ioutil" 
    "path/filepath" 
    "fmt" 
    "log" 
    "net/http"
    "mime"
   _  "bytes"
   _  "crypto/hmac"
   _  "crypto/sha256"
   _  "encoding/base64"
   _  "net/url"
    "os"
    "os/exec"
    _"reflect"
    _ "net/http/httputil"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/awsutil"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/gin-gonic/gin"
    mgo "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"   
    "github.com/line/line-bot-sdk-go/linebot"
)
// Personに共通なものはここにまとめたいですよね
type User struct {
    LineUserId  string        `bson:"line_user_id"`
}
func db_name() string {
	if os.Getenv("GO_ENV") == "development" {
    return  "db"
  } else {
    return os.Getenv("MONGODB_URI")
  }
}
func main() {
  // DB接続
  db_session, err := mgo.Dial(db_name())
  if err != nil {
    panic(err)
  }
  defer db_session.Close()
  users_collection := db_session.DB("heroku_p7vp2k63").C("users")
  users := []User{}
  users_collection.Find(bson.M{}).All(&users)
  for _, user := range users {
    fmt.Println("user_id:", user.LineUserId)
  }
  fmt.Println(filepath.Join(filepath.Dir(os.Args[0]), "line-bot"))
  port := os.Getenv("PORT")
  log.Printf(port)

  if port == "" {
    log.Fatal("$PORT must be set") 
  }
	downloadDir := filepath.Join(filepath.Dir(os.Args[0]), "line-bot")
	_, err = os.Stat(downloadDir)
	if err != nil {
		if err := os.Mkdir(downloadDir, 0777); err != nil {
      log.Fatal("$PORT must be set") 
		}
	}
  router := gin.New()
  router.Use(gin.Logger())
  router.LoadHTMLGlob("templates/*.tmpl.html")
  router.Static("/static", "static")
  exts, _ :=  mime.ExtensionsByType("image/jpeg")

  for _, ext := range exts {
    log.Printf("%s", ext)
  }

  router.GET("/", func(c *gin.Context) {
    c.HTML(http.StatusOK, "index.tmpl.html", nil)
  })

  //この処理を追記
  router.POST("/callback", func(c *gin.Context) {
    log.Printf(c.Request.URL.Path)
    r := c.Request
    log.Printf("ヘッダ %s", r.Header.Get("X-Line-Signature"))
    // Channel ID、Channel Secret、MIDはLINE Developers画面のBasic infomationで確認できます。
    bot, err := linebot.New("key", "ecret")
    events, err := bot.ParseRequest(c.Request)
    if err != nil {
      if err == linebot.ErrInvalidSignature {
        log.Printf("あれ??？")
        fmt.Println(err)
      }
      return
    }
    for _, event := range events {
      if event.Type == linebot.EventTypeFollow {
        source := event.Source
        switch source.Type {
        case linebot.EventSourceTypeUser:
          line_user_id := source.UserID
          log.Printf("followed! %s", line_user_id)
          err = users_collection.Insert(&User{line_user_id})
          if err != nil {
            panic(err)
          }
          if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("友達追加ありがとうございます！\nここから挙式、披露宴中の様子を投稿していただければ幸いです。 ")).Do(); err != nil {
            log.Print(err)
          }
        }
      }
      if event.Type == linebot.EventTypeUnfollow {
        source := event.Source
        switch source.Type {
        case linebot.EventSourceTypeUser:
          line_user_id := source.UserID
          log.Printf("unfollowed! %s", line_user_id)
          selector := bson.M{"line_user_id": line_user_id}
          err := users_collection.Remove(selector)
          if err != nil {
            if v, ok := err.(*mgo.LastError); ok {
              fmt.Printf("Code:%d N:%d UpdatedExisting:%t WTimeout:%t Waited:%d \n", v.Code, v.N, v.UpdatedExisting, v.WTimeout, v.Waited)
            } else {
              fmt.Printf("%+v \n", err)
            }
          }
        }
      }

      if event.Type == linebot.EventTypeMessage {
        switch message := event.Message.(type) {
        case *linebot.TextMessage:
          //if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
          //  log.Print(err)
          //}
          //if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("平岡さんだよ!" + message.Text + "と言いましたね？")).Do(); err != nil {
          //  log.Print(err)
          //}
        //画像が投稿された
        case *linebot.ImageMessage:
          //if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.Text)).Do(); err != nil {
          //  log.Print(err)
          //}
          content, err := bot.GetMessageContent(message.ID).Do()
          if err != nil {
            log.Print(err)
          }
          exts, _ :=  mime.ExtensionsByType(content.ContentType)
          cre := credentials.NewStaticCredentials(
            "XXXXXXXXXXXXXXXXXX",
            "YYYYYYYYYYYYYYYYY",
            "")
            //io.Copy(buffer, content.Content)
            file, err := ioutil.TempFile(downloadDir, "")
            thepath, path_err := filepath.Abs(filepath.Dir(file.Name()))

            log.Print("ファイルをパス ")
            if path_err != nil {
            log.Print(path_err)
            }
            log.Print(thepath)
            if err != nil {
            log.Print(err)
            }
            defer file.Close()
            fstat, err := file.Stat()
            if err != nil {
              panic(file)
            }
            _, err = io.Copy(file, content.Content)
            if err != nil {
              log.Print(err)
            }
            log.Printf("Saved %s", file.Name())
            cli := s3.New(session.New(), &aws.Config{
              Credentials: cre,
              Region:      aws.String("us-west-2"),
            })

            image_key := "/bridal/images/original/"+ message.ID + exts[0]
            put_result, put_err := cli.PutObject(&s3.PutObjectInput{
              Bucket: aws.String("collabostarter"),
              Key:    aws.String(image_key),
              ACL:         aws.String(s3.BucketCannedACLPublicRead),
              Body:   file,
              //Body:   bytes.NewReader(buffer),
              ContentType: aws.String(content.ContentType),
              //ContentLength: aws.Int64(content.ContentLength),
              ContentLength: aws.Int64(fstat.Size()),
            })
            if put_err != nil {
              // Openエラー処理
              log.Print(put_err)
            }
            fmt.Println(awsutil.StringValue(put_result))
            image_url := "https://s3-us-west-2.amazonaws.com/collabostarter" + image_key
            previewImagePath := file.Name() + "-preview"
            log.Print("excecuting convert ...")
            _, err = exec.Command("convert", "-resize", "240x", "jpeg:"+file.Name(), "jpeg:"+previewImagePath).Output()
            if err != nil {
              log.Print(err)
            }
            log.Print("finisehd convert ..." + downloadDir)
            preview_file, p_err := os.Open(previewImagePath) 
            if p_err != nil {
              // Openエラー処理
              log.Print(p_err)
            }
            image_key = "/bridal/images/preview/"+ message.ID + exts[0]
            put_result, put_err = cli.PutObject(&s3.PutObjectInput{
              Bucket: aws.String("collabostarter"),
              Key:    aws.String(image_key),
              ACL:         aws.String(s3.BucketCannedACLPublicRead),
              Body:   preview_file,
              //Body:   bytes.NewReader(buffer),
              ContentType: aws.String(content.ContentType),
              //ContentLength: aws.Int64(content.ContentLength),
              ContentLength: aws.Int64(fstat.Size()),
            })
            preview_image_url := "https://s3-us-west-2.amazonaws.com/collabostarter" + image_key
            defer preview_file.Close()
            defer content.Content.Close()
            //if _, put_err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("画像を投稿くださりありがとうございます！")).Do(); err != nil {
            //  log.Print(put_err)
            //}
            users := []User{}
            users_collection.Find(bson.M{}).All(&users)
            for _, user := range users {
              fmt.Println("sent user_id from :", event.Source.UserID)
              fmt.Println("target user_id:", user.LineUserId)
              if user.LineUserId != event.Source.UserID {
                if _, err := bot.PushMessage(user.LineUserId, linebot.NewImageMessage(image_url, preview_image_url)).Do(); err != nil {
                  log.Print(err)
                }
              } else {
                fmt.Println("same user is not target", user.LineUserId)
              }
            }
          }
        }
      }
    })

    router.Run(":" + port)
  }


func save_s3() {
}
