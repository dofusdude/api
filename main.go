package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/dofusdude/doduapi/ui"
	"github.com/hashicorp/go-memdb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

func AutoUpdate(done chan bool, version *VersionT, updateHook chan bool, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]SearchIndexes) {
	indexWaiterDone := make(chan bool)
	for {
		select {
		case <-done:
			indexWaiterDone <- true
			return
		case <-updateHook:
			var err error
			updateStart := time.Now()
			log.Print("Initialize update...")
			db, idx := IndexApiData(indexWaiterDone, version)

			// send data to main thread
			updateDb <- db
			log.Info("updated db")

			nowOldItemsTable := fmt.Sprintf("%s-all_items", CurrentRedBlueVersionStr(version.MemDb))
			nowOldSetsTable := fmt.Sprintf("%s-sets", CurrentRedBlueVersionStr(version.MemDb))
			nowOldMountsTable := fmt.Sprintf("%s-mounts", CurrentRedBlueVersionStr(version.MemDb))
			nowOldRecipesTable := fmt.Sprintf("%s-recipes", CurrentRedBlueVersionStr(version.MemDb))

			version.MemDb = !version.MemDb // atomic version switch
			log.Info("updated db version")

			delOldTxn := db.Txn(true)
			_, err = delOldTxn.DeleteAll(nowOldItemsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldSetsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldMountsTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			_, err = delOldTxn.DeleteAll(nowOldRecipesTable, "id")
			if err != nil {
				log.Fatal(err)
			}
			delOldTxn.Commit()

			// ----
			updateSearchIndex <- idx

			client := CreateMeiliClient()
			nowOldRedBlueVersion := CurrentRedBlueVersionStr(version.Search)

			version.Search = !version.Search // atomic version switch

			log.Info("changed Meili index")
			for _, lang := range Languages {
				nowOldItemIndexUid := fmt.Sprintf("%s-all_items-%s", nowOldRedBlueVersion, lang)
				nowOldSetIndexUid := fmt.Sprintf("%s-sets-%s", nowOldRedBlueVersion, lang)
				nowOldMountIndexUid := fmt.Sprintf("%s-mounts-%s", nowOldRedBlueVersion, lang)

				itemDeleteTask, err := client.DeleteIndex(nowOldItemIndexUid)
				if err != nil {
					log.Fatal(err)
				}
				_, err = client.WaitForTask(itemDeleteTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}

				setDeletionTask, err := client.DeleteIndex(nowOldSetIndexUid)
				if err != nil {
					log.Fatal(err)
				}
				_, err = client.WaitForTask(setDeletionTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}

				mountDeletionTask, err := client.DeleteIndex(nowOldMountIndexUid)
				if err != nil {
					log.Fatal(err)
				}
				_, err = client.WaitForTask(mountDeletionTask.TaskUID)
				if err != nil {
					log.Fatal(err)
				}
			}
			log.Info("deleted old in-memory data")
			log.Print("Updated", "s", time.Since(updateStart).Seconds())
		}
	}
}

func Hook(updaterDone chan bool, updateDb chan *memdb.MemDB, updateSearchIndex chan map[string]SearchIndexes) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	allDone := false
	go func() {
		for !allDone {
			select {
			case Db = <-updateDb: // override main memory with updated data
			case Indexes = <-updateSearchIndex:
			case <-sigs:
				log.Debug("stopping update routine")
				updaterDone <- true // signal update to stop
				log.Debug("stopped")

				err := httpDataServer.Close()
				if err != nil {
					log.Fatal(err)
				} // close http connections and delete server

				if PrometheusEnabled {
					err = httpMetricsServer.Close()
					if err != nil {
						log.Fatal(err)
					}
				}

				allDone = true
				done <- true
			}
		}
	}()

	<-done
}

func isChannelClosed[T any](ch chan T) bool {
	select {
	case _, ok := <-ch:
		if !ok {
			return true
		}
	default:
	}

	return false
}

var httpDataServer *http.Server
var httpMetricsServer *http.Server
var UpdateChan chan bool

var (
	rootCmd = &cobra.Command{
		Use:           "doduapi",
		Short:         "doduapi – The Dofus encyclopedia API.",
		Long:          ``,
		SilenceErrors: true,
		SilenceUsage:  false,
		Run:           rootCommand,
	}
)

func main() {
	ReadEnvs()

	rootCmd.PersistentFlags().Bool("headless", false, "Run without a TUI.")
	rootCmd.PersistentFlags().Bool("full-img", false, "Load images in prerendered resolutions (~2.5 GB).")

	err := rootCmd.Execute()
	if err != nil && err.Error() != "" {
		fmt.Fprintln(os.Stderr, err)
	}
}

func rootCommand(ccmd *cobra.Command, args []string) {
	var err error
	headless, err := ccmd.Flags().GetBool("headless")
	if err != nil {
		log.Fatal(err)
	}

	FullImg, err = ccmd.Flags().GetBool("full-img")
	if err != nil {
		log.Fatal(err)
	}

	updaterDone := make(chan bool)
	indexWaiterDone := make(chan bool)

	feedbackChan := make(chan string, 5)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ui.Spinner("Init", feedbackChan, false, headless)
		if !isChannelClosed(feedbackChan) {
			close(feedbackChan)
		}
	}()

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Images"
	err = DownloadImages()
	if err != nil {
		log.Fatal(err)
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Persistence"
	err = LoadPersistedElements()
	if err != nil {
		log.Fatal(err)
	}

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Database"
	Db, Indexes = IndexApiData(indexWaiterDone, &Version)
	Version.Search = !Version.Search
	Version.MemDb = !Version.MemDb

	updateDb := make(chan *memdb.MemDB)
	updateSearchIndex := make(chan map[string]SearchIndexes)
	UpdateChan = make(chan bool)

	if isChannelClosed(feedbackChan) {
		os.Exit(1)
	}
	feedbackChan <- "Servers"

	httpDataServer = &http.Server{
		Addr:    fmt.Sprintf(":%s", ApiPort),
		Handler: Router(),
	}

	apiPort, _ := strconv.Atoi(ApiPort)

	if PrometheusEnabled {
		httpMetricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", apiPort+1),
			Handler: promhttp.Handler(),
		}

		go func() {
			if err := httpMetricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
		}()
	}

	go func() {
		if err := httpDataServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go AutoUpdate(updaterDone, &Version, UpdateChan, updateDb, updateSearchIndex)

	if !isChannelClosed(feedbackChan) {
		close(feedbackChan)
	}
	wg.Wait()

	if PrometheusEnabled {
		log.Print("Listening...", "doduapi", apiPort, "metrics", apiPort+1)
	} else {
		log.Print("Listening...", "doduapi", apiPort)
	}

	Hook(updaterDone, updateDb, updateSearchIndex) // block and wait for signal, handle db updates
}
