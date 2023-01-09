package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"

	"github.com/gin-gonic/gin"
)

var err error

type Email struct {
	Email       string `json:"email"`
	Timestamp   int64  `json:"timestamp"`
	Event       string `json:"event"`
	SgEventId   string `json:"sg_event_id"`
	SgMessageId string `json:"sg_message_id"`
	Response    string `json:"response"`
	Reason      string `json:"reason"`
}

type handler struct {
	DB *sql.DB
}

type PostgreConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
}

type MyConfig struct {
	Postgres *PostgreConfig
}

func Init() *sql.DB {
	doc, err := os.ReadFile("config.toml")
	if err != nil {
		panic(err)
	}

	var cfg MyConfig

	err = toml.Unmarshal(doc, &cfg)
	if err != nil {
		panic(err)
	}

	stringConn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Dbname)

	db, err := sql.Open("postgres", stringConn)
	if err != nil {
		log.Fatal("Erro ao realizar conexão")
	} else {
		fmt.Println("Conectado")
	}
	fmt.Println(stringConn)

	db.Ping()
	return db

}
func OpenDB(DB *sql.DB) handler {
	return handler{DB}
}
func (h handler) sendgridWeb(c *gin.Context) {
	//-- DAR ERRO QUANDO O BODY NÃO ESTIVER PREENCHIDO --//

	data, _ := io.ReadAll(c.Request.Body)
	var ListaEventoEmail []Email
	if err := json.Unmarshal(data, &ListaEventoEmail); err != nil {

		log.Fatal("Erro ao deserealizar conteudo contido no Body")

	}
	for _, email := range ListaEventoEmail {
		email.SgMessageId = email.SgMessageId[:strings.IndexByte(email.SgMessageId, '.')]

		//fmt.Println(email.Email, email.Event, email.SgMessageId, email.Timestamp, email.Reason, email.Response)

		//-- PARA CADA EVENTO EXISTE UMA AÇÃO (update) --//

		//-- PROCESSED --//

		if email.Event == "processed" {

			inserting := `insert into email (email_destinatario, enviado, id_sendgrid) values ($1, $2, $3)`
			_, err := h.DB.Exec(inserting, email.Email, true, email.SgMessageId)
			if err != nil {
				log.Fatal("Erro ao inserir dados na tabela email")
			}

		}
		//-- DELIVERED --//

		if email.Event == "delivered" {
			_, err := h.DB.Exec(`UPDATE email SET recebido=true WHERE id_sendgrid=$1`, email.SgMessageId)
			if err != nil {
				log.Fatal("Erro ao realizar upload no email com evento DELIVERED")
			}

			//-- OPEN --//

		} else if email.Event == "open" {
			_, err := h.DB.Exec(`UPDATE email SET recebido=true, aberto=true WHERE id_sendgrid=$1`, email.SgMessageId)
			if err != nil {
				log.Fatal("Erro ao realizar upload no email com evento OPEN")
			}
		} else if email.Event == "click" {
			_, err := h.DB.Exec(`UPDATE email SET click = click + 1 WHERE id_sendgrid=$1`, email.SgMessageId)
			if err != nil {
				log.Fatal("Erro ao realizar upload no email com evento CLICK")
			}
		}
		//-- ARMAZENAMENTO DE ERROS --//

		/*if email.Event == "dropped" || email.Event == "bounce" || email.Event == "deferred" {

			var reason = email.Reason
			fmt.Println(reason)
			var response = email.Response

			if email.Reason == "" {
				reason = email.Response
			} else {
				reason = email.Reason
			}

			if email.Response == "" {
				response = email.Reason
			} else {
				response = email.Response
			}

			var sgID = email.SgMessageId

			var teste = fmt.Sprintf(`update email set erro = concat(erro,' ','%s') where id_sendgrid='%s'`, response, sgID)
			fmt.Println(teste)

			_, err := h.DB.Exec(teste)
			if err != nil {
				log.Fatal("Erro ao realizar upload no email com evento ERROR")

			}
			fmt.Println(email.Reason, email.SgMessageId, email.Response)

		}
		fmt.Println(email.Reason)*/

	}
}

/*func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"Message": "pong",
	})
}*/

func main() {
	dbload := Init()
	h := OpenDB(dbload)

	/*port := os.Getenv("HTTP_PLATFORM_PORT")
	if port == "" {
		port = "8080"
	}*/

	router := gin.Default()
	router.POST("/webhook/sendgrid", h.sendgridWeb)
	//router.GET("/ping", ping)
	port := os.Getenv("HTTP_PLATFORM_PORT")

	// default back to 8080 for local dev
	if port == "" {
		port = "8080"
	}

	router.Run("127.0.0.1:" + port)

}
