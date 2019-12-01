package repository

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"strings"
)

const dbDialect = "mysql"
const dbPath = "b641d7e50b5242:0708f315@(us-cdbr-iron-east-05.cleardb.net)/heroku_07b6f05f1995915"

type GameParticipants struct {
	ID      int `gorm:"primary_key"`
	Plus    string
	Minus   string
	Injured string
}

func (participants *GameParticipants) plus(name string) {
	participants.Plus = addParticipant(participants.Plus, name)
}

func (participants *GameParticipants) minus(name string) {
	participants.Minus = addParticipant(participants.Minus, name)
}

func (participants *GameParticipants) injured(name string) {
	participants.Injured = addParticipant(participants.Injured, name)
}

func addParticipant(list string, name string) string {
	if list != "" {
		list += ","
	}
	list += name
	return list
}

func (participants *GameParticipants) deleteDuplicates( name string) {
	participants.Plus = deleteDuplicate(participants.Plus, name)
	participants.Minus = deleteDuplicate(participants.Minus, name)
	participants.Injured = deleteDuplicate(participants.Injured, name)
}

func deleteDuplicate(players string, name string) string {
	split := strings.Split(players, ",")

	for i, existingName := range split {
		if strings.ToLower(existingName) == strings.ToLower(name) {
			split = append(split[:i], split[i+1:]...)
			break
		}
	}
	return strings.Join(split, ",")
}

func openDb() (*gorm.DB, error) {
	db, err := gorm.Open(dbDialect, dbPath)
	return db, err
}

func findLastList(list *GameParticipants) error {
	db, err := openDb()
	if err != nil {
		return err
	}
	defer db.Close()

	db.Debug().Last(list)
	fmt.Println(&list)
	return nil
}

func findLastListWithDb(db *gorm.DB, list *GameParticipants) error {

	db.Debug().Last(list)
	fmt.Println(&list)
	return nil
}

// public methods

func InitialMigration() {
	db, err := openDb()
	if err != nil {
		fmt.Println(err.Error())
		panic("Fault occured while migrating")
	}
	defer db.Close()
	db.AutoMigrate(&GameParticipants{})
	fmt.Println("Initial migration completed.")
}

func CreateNewList() error {
	db, err := openDb()
	if err != nil {
		return err
	}
	defer db.Close()

	lastList := &GameParticipants{}
	if errToFind := findLastListWithDb(db, lastList); errToFind != nil {
		return errToFind
	}

	db.Debug().Create(&GameParticipants{Injured: lastList.Injured})

	return nil
}

func AddNewParticipant(name string, action string) error {
	db, err := openDb()
	if err != nil {
		return err
	}
	defer db.Close()

	var lastList GameParticipants
	if findError := findLastListWithDb(db, &lastList); findError != nil {
		return findError
	}

	lastList.deleteDuplicates(name)

	switch action {
		case "+":
			lastList.plus(name)
		case "-":
			lastList.minus(name)
		case "!":
			lastList.injured(name)
		default:
			return fmt.Errorf("Неизвестная команда ")
	}

	db.Debug().Save(&lastList)

	return nil
}

func FindLastList() (*GameParticipants, error) {
	var list GameParticipants
	if findErr := findLastList(&list); findErr != nil {
		return nil, findErr
	}
	return &list, nil
}
