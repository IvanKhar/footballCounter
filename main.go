package main

import (
	"fmt"
	"footballCounter/repository"
	tgbotapi "github.com/Syfaro/telegram-bot-api"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"os"
	"reflect"
	"strings"
)

const greetingMsgText = "Пожалуйста, выберете, что бы Вы хотели сделать:"
const greetingMsgTextPost = "%s\n\nПожалуйста, выберете, что бы Вы хотели сделать:"
const infoMsgText = "Справочная информация: данный бот предназначен для упрощения записи на игру. \nЧтобы отметить, что Вы хотите пойти на игру, отправьте в чат \"+\". \nЕсли точно не получится придти на ближайшую игру, то отправьте \"-\". \nЕсли Вы травмированы, то отправьте \"!\"."

func main() {
	repository.InitialMigration()
	//Calling bot
	telegramBot()
}

var botToken = os.Getenv("TBOT_TOKEN")

func telegramBot() {

	fmt.Println("BOT token ", botToken)
	//Creating bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		panic(err)
	}

	fmt.Println("Connected to telegram.")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	processRequests(bot, updateConfig)
}

func processRequests(bot *tgbotapi.BotAPI, updateConfig tgbotapi.UpdateConfig) {

	updatesChannel, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		panic(err)
	}

	for update := range updatesChannel {

		var msg tgbotapi.MessageConfig

		if update.CallbackQuery != nil {
			msg = processCallBackQuery(update)
			bot.Send(msg)
			continue
		}

		//Check if we got text message
		if reflect.TypeOf(update.Message.Text).Kind() == reflect.String && update.Message.Text != "" {
			msg = getStartMessageWithKeyBoard(update.Message.Chat.ID, "Привет!")
		} else {
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Пока умею принимать только текстовые сообщения.")
		}
		bot.Send(msg)
	}
}

func processCallBackQuery(request tgbotapi.Update) tgbotapi.MessageConfig {

	switch request.CallbackQuery.Data {
	case "manage":
		return getManageKeyBoard(request.CallbackQuery.Message.Chat.ID)
	case "list":
		return getStartMessageWithKeyBoard(request.CallbackQuery.Message.Chat.ID, processShowListRequest())
	case "participate":
		return getParticipateKeyboard(request.CallbackQuery.Message.Chat.ID)
	case "newList":
		if err := processNewListRequest(request.CallbackQuery); err != nil {
			return getStartMessageWithKeyBoard(request.CallbackQuery.Message.Chat.ID, err.Error())
		} else {
			return getStartMessageWithKeyBoard(request.CallbackQuery.Message.Chat.ID, "Создал")
		}
	default:
		if err := processAddParticipantRequest(&request); err != nil {
			return getStartMessageWithKeyBoard(request.CallbackQuery.Message.Chat.ID, err.Error())
		} else {
			return getStartMessageWithKeyBoard(request.CallbackQuery.Message.Chat.ID, "Записал ;)")
		}
	}
}

func getManageKeyBoard(chatId int64) tgbotapi.MessageConfig {

	newListButton := tgbotapi.NewInlineKeyboardButtonData("Создать новый список", "newList")
	//editListButton := tgbotapi.NewInlineKeyboardButtonData("Редактироовать существующий", "editList")
	keyboardRow := tgbotapi.NewInlineKeyboardRow(newListButton)

	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf(greetingMsgText))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRow)
	return msg
}

func getStartMessageWithKeyBoard(chatId int64, responseText string) tgbotapi.MessageConfig {

	partButton := tgbotapi.NewInlineKeyboardButtonData("Записаться", "participate")
	listButton := tgbotapi.NewInlineKeyboardButtonData("Узнать расклад", "list")
	keyboardRow1 := tgbotapi.NewInlineKeyboardRow(partButton, listButton)

	manageButton := tgbotapi.NewInlineKeyboardButtonData("Управлять списками", "manage")
	keyboardRow2 := tgbotapi.NewInlineKeyboardRow(manageButton)

	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf(greetingMsgTextPost, responseText))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRow1, keyboardRow2)
	return msg
}

func getParticipateKeyboard(chatId int64) tgbotapi.MessageConfig {

	plus := tgbotapi.NewInlineKeyboardButtonData("Приду", "+")
	minus := tgbotapi.NewInlineKeyboardButtonData("Не приду", "-")
	injured := tgbotapi.NewInlineKeyboardButtonData("Травмирован", "!")
	keyboardRow := tgbotapi.NewInlineKeyboardRow(plus, minus, injured)

	msg := tgbotapi.NewMessage(chatId, "Укажите, сможете ли вы участвовать")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboardRow)

	return msg
}

const errorOccurredText = "Возникла ошибка: %s"

func processNewListRequest(request *tgbotapi.CallbackQuery) error {

	userName := fmt.Sprint(request.From.FirstName, request.From.LastName)
	if strings.Contains(userName, "IvanKharkevich") {
		if err := repository.CreateNewList(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("У вас нет прав на создание нового списка")
	}
	return nil
}

func processShowListRequest() string {

	var message string
	if list, err := repository.FindLastList(); err != nil {
		message = fmt.Sprintf(errorOccurredText, err.Error())
	} else {
		message = fmt.Sprintf(
			"Расклад на ближайшую игру:\n- точно идут %d человек: %s\n- не идут %d человек: %s\n- травмированы %d человек: %s",
			getParticipantCount(&list.Plus), list.Plus,
			getParticipantCount(&list.Minus), list.Minus,
			getParticipantCount(&list.Injured), list.Injured)
	}
	return message
}

func createMsgWithKeyboard(chatId int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.MessageConfig {

	msg := tgbotapi.NewMessage(chatId, text)
	msg.ReplyMarkup = keyboard
	return msg
}

func processAddParticipantRequest(request *tgbotapi.Update) error {

	userFullName := fmt.Sprintf("%s %s", string(request.CallbackQuery.From.LastName), string(request.CallbackQuery.From.FirstName))
	fmt.Println("Имя пользователя", userFullName, "action:", request.CallbackQuery.Data)
	if err := repository.AddNewParticipant(userFullName, string(request.CallbackQuery.Data)); err != nil {
		return err
	}
	return nil
}

func getParticipantCount(s *string) int {
	if *s == "" {
		return 0
	} else {
		return len(strings.Split(*s, ","))
	}
}
