package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type step int

const (
	zero step = iota
	pillsLeft
	pillsAll
	bestTime
	notify
	finish
)

type tgbot struct {
	bot        *tgbotapi.BotAPI `json:"-"`
	Step       step             `json:"step"`
	PillsLeft  int              `json:"left"`
	PillsAll   int              `json:"all"`
	Hour       int              `json:"hour"`
	SweetNames []string         `json:"names"`
}

func main() {
	botAPI, err := tgbotapi.NewBotAPI(os.Getenv("TG_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}

	botAPI.Debug = true

	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	bt := os.Getenv("SWEET_NAMES")

	bot := tgbot{
		bot:  botAPI,
		Step: zero,
	}

	err = json.Unmarshal([]byte(bt), &bot.SweetNames)
	if err != nil {
		log.Fatalln(err)
	}

	bot.run()
}

func (b *tgbot) randName() string {
	return b.SweetNames[rand.Intn(len(b.SweetNames)-1)]
}

func (b *tgbot) run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}
		if b.Step == pillsLeft {
			msg, err := b.handlePillsLeft(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			if msg != nil {
				b.bot.Send(msg)
				continue
			}
		}
		if b.Step == pillsAll {
			msg, err := b.handlePillsAll(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			if msg != nil {
				b.bot.Send(msg)
				continue
			}
		}
		if b.Step == bestTime {
			msg, err := b.handleTime(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			if msg != nil {
				b.bot.Send(msg)
			}

		}

		if b.Step == notify {
			go b.startNotifyWorker(update)
		}

		if msg := b.handleNewMember(update); msg != nil {
			b.bot.Send(msg)
			continue
		}
		if msg := b.handleCommand(update); msg != nil {
			b.bot.Send(msg)
		}
	}
}

func (b *tgbot) handleNewMember(update tgbotapi.Update) *tgbotapi.MessageConfig {
	if len(update.Message.NewChatMembers) == 0 {
		return nil
	}
	for range update.Message.NewChatMembers {
		text := fmt.Sprintf(
			"Привет %s! Этот телеграм бот поможет тебе принимать "+
				"твои противозачаточные таблетки вовремя. Начни с команды '''/start'''\n", b.randName())
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
		return &msg

	}
	return nil
}

func (b *tgbot) handleCommand(update tgbotapi.Update) *tgbotapi.MessageConfig {
	if !update.Message.IsCommand() { // ignore any non-command Messages
		return nil
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

	switch update.Message.Command() {
	case "start":
		msg.Text = fmt.Sprintf(
			"%s, скажи пожалуйста сколько осталось таблеток в упаковке КОКов?\n", b.randName(),
		)
		b.Step = pillsLeft
	case "reset":
		msg.Text = "Данные сброшены. Начни снова с команды '''/start'''"
		b.Step = zero
	case "help":
		msg.Text = "I understand /start and /reset"
	default:
		msg.Text = "I don't know that command"
	}
	return &msg
}

func (b *tgbot) handlePillsLeft(update tgbotapi.Update) (*tgbotapi.MessageConfig, error) {
	left, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		return nil, errors.New("Нужно число")
	}
	b.PillsLeft = left
	b.Step = pillsAll

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Спасибо %s! А теперь скажи пожалуйста сколько ВСЕГО "+
			"таблеток должно быть в упаковке?\n", b.randName()),
	)
	return &msg, nil
}

func (b *tgbot) handlePillsAll(update tgbotapi.Update) (*tgbotapi.MessageConfig, error) {
	all, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		return nil, errors.New("Нужно число")
	}
	b.PillsAll = all
	b.Step = bestTime

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Спасибо %s! А теперь скажи пожалуйста в какой"+
			" время тебе удобно напоминать о приеме таблеток? Число между 1-24\n", b.randName()),
	)
	return &msg, nil
}

func (b *tgbot) handleTime(update tgbotapi.Update) (*tgbotapi.MessageConfig, error) {
	hour, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		return nil, errors.New("Нужно число между 1-24")
	}
	b.Hour = hour
	b.Step = notify

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Спасибо %s! Напоминания будут приходить каждый %d час\n", b.randName(), b.Hour),
	)
	return &msg, nil
}

var weekdays = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

func (b *tgbot) startNotifyWorker(update tgbotapi.Update) {
	log.Println("WORKER STARTED")
	b.Step = finish
	sentDay := 0
	tick := time.NewTicker(time.Minute * 5)
	for t := range tick.C {
		if sentDay == t.Day() {
			continue
		}
		log.Println("HOURS:", t.Hour(), b.Hour)

		if t.Hour() == b.Hour {
			if b.PillsLeft == 0 {
				b.PillsLeft = b.PillsAll
			}
			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				fmt.Sprintf(
					"Привет %s! Пришло время принять таблетку! Сегодня %s. Осталось таблеток: %d. ",
					b.randName(),
					weekdays[t.Weekday()],
					b.PillsLeft,
				),
			)
			b.PillsLeft--

			if b.PillsLeft < 3 {
				msg.Text = msg.Text + "Таблетки заканчиваются! Не забудь купить новые!"
			}
			b.bot.Send(msg)
			sentDay = t.Day()
		}
	}
}

func (b *tgbot) storeState() error {
	bt, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return os.WriteFile("storage.json", bt, 0644)
}

func (b *tgbot) loadState() error {
	bt, err := os.ReadFile("storage.json")
	if err != nil {
		log.Fatalln(err)
	}
	return json.Unmarshal(bt, b)

}
