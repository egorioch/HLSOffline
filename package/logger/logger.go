package logger

import (
	"log"
	"os"
)

func InitLogger() {
	logfile, err := os.OpenFile("package/static/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}

	log.SetOutput(logfile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}
