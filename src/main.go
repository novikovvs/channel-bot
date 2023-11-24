package main

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v10/orm"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/go-pg/pg/v10"
)

var threadId int64
var channelId int64

type User struct {
	tableName struct{} `pg:"users"`
	ID        int
	TgUserId  int `pg:"tg_user_id,unique"`
}

func initialization() {

	ctx := context.Background()

	sosiskaDB := pg.Connect(&pg.Options{
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Database: os.Getenv("POSTGRES_DB"),
	})
	defer sosiskaDB.Close()

	if err := sosiskaDB.Ping(ctx); err != nil {
		panic(err)
	}

	err := createSchema(sosiskaDB)

	if err != nil {
		panic(err)
	}

	log.Println("Migrated")
}

func createSchema(db *pg.DB) error {
	models := []interface{}{
		(*User)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
		})
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func main() {
	err := godotenv.Load("../.env")

	if err != nil {
		log.Println("Error loading .env file")
	}

	initialization()

	if channelId, err = strconv.ParseInt(os.Getenv("CHANNEL_ID"), 10, 64); err != nil {
		fmt.Println("Dont look channel")
	}

	if threadId, err = strconv.ParseInt(os.Getenv("THREAD_ID"), 10, 64); err != nil {
		fmt.Println("Dont look thread")
	}

	token := os.Getenv("TOKEN")

	if token == "" {
		panic("TOKEN environment variable is empty")
	}

	b, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{},
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: gotgbot.DefaultTimeout,
				APIURL:  gotgbot.DefaultAPIURL,
			},
		},
	})

	if err != nil {
		panic("failed to create new bot: " + err.Error())
	}

	updater := ext.NewUpdater(&ext.UpdaterOpts{
		Dispatcher: ext.NewDispatcher(&ext.DispatcherOpts{
			Error: func(b *gotgbot.Bot, ctx *ext.Context, err error) ext.DispatcherAction {
				log.Println("an error occurred while handling update:", err.Error())
				return ext.DispatcherActionNoop
			},
			MaxRoutines: ext.DefaultMaxRoutines,
		}),
	})

	dispatcher := updater.Dispatcher

	dispatcher.AddHandler(handlers.NewCommand("start", startCommandHandler))
	dispatcher.AddHandler(handlers.NewMessage(message.All, echo))

	err = updater.StartPolling(b, &ext.PollingOpts{
		DropPendingUpdates: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 9,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: time.Second * 10,
			},
		},
	})

	if err != nil {
		panic("failed to start polling: " + err.Error())
	}

	log.Printf("%s has been started...\n", b.User.Username)

	updater.Idle()
}

func echo(b *gotgbot.Bot, ctx *ext.Context) error {
	if channelId == ctx.EffectiveChat.Id {
		return nil
	}

	chatMember, _ := b.GetChatMember(channelId, ctx.EffectiveUser.Id, nil)

	if chatMember == nil {
		log.Println("User not in channel")
		return nil
	}

	_, err := b.ForwardMessage(channelId, ctx.EffectiveChat.Id, ctx.Message.MessageId,
		&gotgbot.ForwardMessageOpts{
			MessageThreadId: threadId,
		})

	if err != nil {
		return fmt.Errorf("failed to echo message: %w", err)
	}
	return nil
}
func startCommandHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	chatMember, _ := b.GetChatMember(channelId, ctx.EffectiveUser.Id, nil)

	if chatMember == nil {
		return nil
	}

	sosiskaDB := pg.Connect(&pg.Options{
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		Database: os.Getenv("POSTGRES_DB"),
	})
	defer sosiskaDB.Close()

	_, err := sosiskaDB.Model(&User{TgUserId: int(ctx.EffectiveUser.Id)}).Insert()
	if err != nil {
		return err
	}

	_, err = b.SendMessage(ctx.EffectiveMessage.Chat.Id, "Добро пожаловать в клуб, сладeнький носитель сосиски!", nil)
	if err != nil {
		return err
	}

	return nil
}
