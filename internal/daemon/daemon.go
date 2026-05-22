//go:build gtk

package daemon

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/hotkey"
	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/ui"
	"github.com/chriscast/chriscast/internal/watcher"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func gtkQuit() { gtk.MainQuit() }

type Daemon struct {
	cfg *config.Config
	ix  *index.Index
	app *ui.App
	w   *watcher.Watcher
	hk  *hotkey.Listener
}

func New(cfg *config.Config, ix *index.Index) (*Daemon, error) {
	app, err := ui.NewApp(cfg, ix)
	if err != nil {
		return nil, err
	}
	app.SetDaemon(true)
	return &Daemon{cfg: cfg, ix: ix, app: app}, nil
}

func (d *Daemon) Run() error {
	// fsnotify watcher
	w, err := watcher.New(d.cfg, d.ix)
	if err != nil {
		return err
	}
	d.w = w
	go func() {
		if err := w.Run(); err != nil {
			log.Printf("watcher exited: %v", err)
		}
	}()

	// hotkey listener
	hk, err := hotkey.New(d.cfg.Hotkey, func() {
		glib.IdleAdd(func() bool {
			d.app.Show()
			return false
		})
	})
	if err != nil {
		log.Printf("hotkey: %v (UI still launchable via `chriscast ui`)", err)
	} else {
		d.hk = hk
		go hk.Run()
	}

	// periodic save (in case daemon dies)
	go d.periodicSave(30 * time.Second)

	// signal handling — graceful save on shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutdown: saving index")
		if err := d.ix.Save(); err != nil {
			log.Printf("save: %v", err)
		}
		// schedule clean GTK quit on main loop; Main() returns, Run() exits
		glib.IdleAdd(func() bool {
			gtkQuit()
			return false
		})
	}()

	log.Printf("daemon running (hotkey=%s, %d files indexed)", d.cfg.Hotkey, d.ix.Store.Len())
	d.app.Main()
	return nil
}

func (d *Daemon) periodicSave(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		if err := d.ix.Save(); err != nil {
			log.Printf("periodic save: %v", err)
		}
	}
}
