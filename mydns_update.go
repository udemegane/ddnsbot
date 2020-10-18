package main

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func ticker(ctx context.Context) error {
	t := time.NewTicker(3600 * 12 * time.Second) //1秒周期の ticker
	defer t.Stop()

	for {
		select {
		case now := <-t.C:
			fmt.Println("IPaddr update")
			fmt.Println(now.Format(time.RFC3339))
			cmd := exec.Command("./update.sh")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				sendmail("error by mydns_update", "mydnsのIP更新が止まった")
				panic(err)
			}

		case <-ctx.Done():
			fmt.Println("Stop child")
			return ctx.Err()
		}
	}
}

func SignalContext(ctx context.Context) context.Context {
	parent, cancelParent := context.WithCancel(ctx)
	go func() {
		defer cancelParent()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)
		defer signal.Stop(sig)

		select {
		case <-parent.Done():
			fmt.Println("Cancel from parent")
			return
		case s := <-sig:
			switch s {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				sendmail("error by mydns_update", "mydnsのIP更新が止まった")
				fmt.Println("Stop!")
				return
			}
		}
	}()

	return parent
}

func Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	defer close(errCh)

	parent := SignalContext(ctx)
	go func() {
		child, cancelChild := context.WithTimeout(parent, 36000*time.Second)
		defer cancelChild()
		errCh <- ticker(child)
	}()

	err := <-errCh
	fmt.Println("Done parent")
	return err
}

func sendmail(subject, message string) error {
	mailaddr := os.Getenv("GMAIL_MAIN")
	pwd := os.Getenv("GMAIL_MAIN_PW")
	auth := smtp.PlainAuth(
		"",
		mailaddr,
		pwd,
		"smtp.gmail.com",
	)

	return smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		mailaddr,           // 送信元
		[]string{mailaddr}, // 送信先
		[]byte(
			"To: "+mailaddr+"\r\n"+
				"Subject:"+subject+"\r\n"+
				"\r\n"+
				message),
	)
}

func main() {
	ctx := context.Background()
	if err := Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("Done")
}
