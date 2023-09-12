package main

import (
	"errors"
	"fmt"
	"github.com/filinvadim/vadim-bot/pkg"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type step int

const (
	zero step = iota
	dragName
	pillsLeft
	pillsAll
	bestTime
	anotherDrug
	notify
	finish
)

type tgbot struct {
	bot         *tgbotapi.BotAPI
	Step        step
	Drugs       []pkg.Drug
	drugNames   map[string]struct{}
	SweetNames  []string
	updatesChan chan tgbotapi.Update
}

func main() {
	botAPI, err := tgbotapi.NewBotAPI(os.Getenv("TG_TOKEN"))
	if err != nil {
		log.Fatalln(err)
	}
	botAPI.Debug = false

	log.Printf("Authorized on account %s", botAPI.Self.UserName)

	bot := tgbot{
		bot:         botAPI,
		updatesChan: make(chan tgbotapi.Update, 10),
		SweetNames:  pkg.SweetNames,
		drugNames:   map[string]struct{}{},
	}

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("UP"))
		})
		http.ListenAndServe(":8080", mux)
	}()

	bot.run()
}

func (b *tgbot) randName() string {
	return b.SweetNames[rand.Intn(len(b.SweetNames)-1)]
}

const (
	modalAnother = "Добавить еще одно лекарство"
	modalNext    = "Дальше"
)

func (b *tgbot) run() {
	go func() {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		updates := b.bot.GetUpdatesChan(u)
		for u := range updates {
			b.updatesChan <- u
		}
	}()

	modal := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(modalAnother),
			tgbotapi.NewKeyboardButton(modalNext),
		),
	)

	for update := range b.updatesChan {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}
		switch b.Step {
		case zero:
			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				fmt.Sprintf("%s, напиши пожалуйста имя лекарства", b.randName()),
			)
			b.Step = dragName
			b.bot.Send(msg)
		case dragName:
			if _, ok := b.drugNames[update.Message.Text]; ok {
				msg := tgbotapi.NewMessage(
					update.Message.Chat.ID,
					fmt.Sprintf("%s, это имя уже есть в списке. Попробуй ввести другое", b.randName()),
				)
				b.bot.Send(msg)
				continue
			}

			b.Drugs = append(b.Drugs, pkg.Drug{
				Name:          update.Message.Text,
				PillTakenTime: time.Time{},
			})
			b.drugNames[update.Message.Text] = struct{}{}

			b.Step = pillsLeft
			b.bot.Send(
				tgbotapi.NewMessage(
					update.Message.Chat.ID,
					fmt.Sprintf(pkg.PillsLeftStageText, b.randName()),
				),
			)
		case pillsLeft:
			msg, err := b.handlePillsLeft(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			b.bot.Send(msg)
			b.Step = pillsAll

		case pillsAll:
			msg, err := b.handlePillsAll(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			b.bot.Send(msg)
			b.Step = bestTime

		case bestTime:
			msg, err := b.handleTime(update)
			if err != nil {
				b.bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}

			msg.ReplyMarkup = modal
			b.bot.Send(msg)
			b.Step = anotherDrug

		case anotherDrug:
			switch update.Message.Text {
			case modalAnother:
				b.Step = zero
			case modalNext:
				b.Step = notify
			default:
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Нужно нажать кнопку")
				msg.ReplyMarkup = modal
				b.bot.Send(msg)
				continue
			}
			b.updatesChan <- update

		case notify:
			b.Step = finish
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(pkg.NotifyStageText, b.randName()))
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)

			for _, d := range b.Drugs {
				msg.Text += fmt.Sprintf("Лекарство %s, уведомление каждый %d час\n", d.Name, d.TakingHour)
			}
			b.bot.Send(msg)
			go b.startNotifyWorker(update)

		default:
			if msg := b.handleCommand(update); msg != nil {
				b.bot.Send(msg)
			}
			if msg := b.handleNewMember(update); msg != nil {
				b.Step = zero
				b.bot.Send(msg)
			}
		}
	}
}

func (b *tgbot) handleNewMember(update tgbotapi.Update) *tgbotapi.MessageConfig {
	if len(update.Message.NewChatMembers) == 0 {
		return nil
	}
	for range update.Message.NewChatMembers {
		text := fmt.Sprintf(pkg.StartStageText, b.randName())
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

	b.Drugs[len(b.Drugs)-1].PillsLeft = left

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf(pkg.PillsTotalStageText, b.randName()),
	)

	return &msg, nil
}

func (b *tgbot) handlePillsAll(update tgbotapi.Update) (*tgbotapi.MessageConfig, error) {
	all, err := strconv.Atoi(update.Message.Text)
	if err != nil {
		return nil, errors.New("Нужно число")
	}

	if b.Drugs[len(b.Drugs)-1].PillsLeft > all {
		return nil, errors.New("Осталось больше, чем есть всего")
	}
	b.Drugs[len(b.Drugs)-1].PillsTotal = all

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf(pkg.BestTimeStageText, b.randName()),
	)
	return &msg, nil
}

func (b *tgbot) handleTime(update tgbotapi.Update) (*tgbotapi.MessageConfig, error) {
	hour, err := strconv.Atoi(update.Message.Text)
	if err != nil || hour > 24 {
		return nil, errors.New("Нужно число между 1-24")
	}

	b.Drugs[len(b.Drugs)-1].TakingHour = hour

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		fmt.Sprintf("Спасибо, %s", b.randName()),
	)
	return &msg, nil
}

func (b *tgbot) startNotifyWorker(update tgbotapi.Update) {
	log.Println("WORKER STARTED")
	tick := time.NewTicker(time.Minute * 5)
	for t := range tick.C {
		for i, d := range b.Drugs {
			if d.IsAlreadyTaken(t) || t.Hour() != d.TakingHour {
				continue
			}

			log.Printf(
				"drug: %s, hour now: %d == taking hour: %d, prev taken date: %s",
				d.Name, t.Hour(), d.TakingHour, d.PillTakenTime.String(),
			)

			d.TakePill()

			msg := tgbotapi.NewMessage(
				update.Message.Chat.ID,
				fmt.Sprintf(
					pkg.PillsTakingTimeText,
					b.randName(),
					d.Name,
					pkg.GetWeekdayName(t),
					d.PillsLeft,
				),
			)

			if d.IsPillsRunOut() {
				msg.Text = msg.Text + pkg.PillsRunOutText
			}
			b.bot.Send(msg)

			b.Drugs[i].PillTakenTime = time.Now()
		}
	}
}
