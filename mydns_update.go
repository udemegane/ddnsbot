package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func ticker(ctx context.Context) error {
	var ipaddr string
	ipaddr = "0.0.0.0"
	t := time.NewTicker(30 * time.Second) //1秒周期の ticker
	defer t.Stop()

	for {
		select {
		case now := <-t.C:
			out, err := exec.Command("curl", "inet-ip.info").Output()
			tmp := string(out)
			fmt.Println(now.Format(time.RFC3339) + "\n" + tmp)
			if ipaddr != tmp {
				ipaddr = tmp
				ipupdater(ipaddr)
			}
			if err != nil {
				sendmail("error by mydns_update", "なんかipが取得できなかった")
				panic(err)
			}

		case <-ctx.Done():
			fmt.Println("Stop child")
			return ctx.Err()
		}
	}
}

func ipupdater(addr string) {
	fmt.Println("new ip address: " + addr)
	cmd := exec.Command("./update.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	sendmail("ipアドレスの変更を検知", "ipが変わった。\nnew ip: "+addr)
	err := cmd.Run()
	if err != nil {
		sendmail("error by mydns_update", "更新中にエラー吐いた")
		panic(err)
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

func sendmail(subject, message string) {
	//あとでつくる
}

func main() {
	ctx := context.Background()
	if err := Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("Done")
}
