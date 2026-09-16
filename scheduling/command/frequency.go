package command

import (
	"fmt"
	"strings"
	"time"
)

func Cron(expr string) func(*Options) {
	return func(o *Options) { o.Crontab = expr }
}

func EveryThirtySeconds() func(*Options) {
	return func(o *Options) {
		o.Crontab = "*/30 * * * * *"
		o.WithSeconds = true
	}
}

func EveryMinute() func(*Options)         { return Cron("* * * * *") }
func EveryTwoMinutes() func(*Options)     { return Cron("*/2 * * * *") }
func EveryFiveMinutes() func(*Options)    { return Cron("*/5 * * * *") }
func EveryTenMinutes() func(*Options)     { return Cron("*/10 * * * *") }
func EveryFifteenMinutes() func(*Options) { return Cron("*/15 * * * *") }
func EveryThirtyMinutes() func(*Options)  { return Cron("*/30 * * * *") }

func Hourly() func(*Options)             { return Cron("0 * * * *") }
func HourlyAt(minute int) func(*Options) { return Cron(fmt.Sprintf("%d * * * *", minute)) }

func Daily() func(*Options) { return Cron("0 0 * * *") }
func DailyAt(t string) func(*Options) {
	parts := strings.Split(t, ":")
	return Cron(fmt.Sprintf("%s %s * * *", parts[1], parts[0]))
}

func TwiceDaily(h1, h2 int) func(*Options) {
	return Cron(fmt.Sprintf("0 %d,%d * * *", h1, h2))
}

func Weekly() func(*Options) { return Cron("0 0 * * 0") }
func WeeklyOn(weekday time.Weekday, t string) func(*Options) {
	parts := strings.Split(t, ":")
	return Cron(fmt.Sprintf("%s %s * * %d", parts[1], parts[0], int(weekday)))
}

func Monthly() func(*Options) { return Cron("0 0 1 * *") }
func MonthlyOn(day int, t string) func(*Options) {
	parts := strings.Split(t, ":")
	return Cron(fmt.Sprintf("%s %s %d * *", parts[1], parts[0], day))
}

func Quarterly() func(*Options) { return Cron("0 0 1 */3 *") }
func Yearly() func(*Options)    { return Cron("0 0 1 1 *") }

func Timezone(tz string) func(*Options) { return func(o *Options) { o.Timezone = tz } }
