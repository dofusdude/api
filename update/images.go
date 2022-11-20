package update

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

func DownloadImagesLauncher(hashJson map[string]interface{}) error {
	main := hashJson["main"].(map[string]interface{})
	files := main["files"].(map[string]interface{})

	wg := sync.WaitGroup{}

	// item bitmaps
	var itemImages0 HashFile
	itemImages0.Filename = "content/gfx/items/bitmap0.d2p"
	itemImages0.FriendlyName = "data/tmp/bitmaps_0.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := DownloadHashImageFileInJson(files, itemImages0); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages1 HashFile
	itemImages1.Filename = "content/gfx/items/bitmap0_1.d2p"
	itemImages1.FriendlyName = "data/tmp/bitmaps_1.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := DownloadHashImageFileInJson(files, itemImages1); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages2 HashFile
	itemImages2.Filename = "content/gfx/items/bitmap1.d2p"
	itemImages2.FriendlyName = "data/tmp/bitmaps_2.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := DownloadHashImageFileInJson(files, itemImages2); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages3 HashFile
	itemImages3.Filename = "content/gfx/items/bitmap1_1.d2p"
	itemImages3.FriendlyName = "data/tmp/bitmaps_3.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := DownloadHashImageFileInJson(files, itemImages3); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages4 HashFile
	itemImages4.Filename = "content/gfx/items/bitmap1_2.d2p"
	itemImages4.FriendlyName = "data/tmp/bitmaps_4.d2p"

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := DownloadHashImageFileInJson(files, itemImages4); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
	path, err := os.Getwd()
	if err != nil {
		return err
	}

	inPath := fmt.Sprintf("%s/data/tmp", path)
	outPath := fmt.Sprintf("%s/data/img/item", path)
	absConvertCmd := fmt.Sprintf("%s/PyDofus/%s_unpack.py", path, "d2p")
	if err := exec.Command("/usr/local/bin/python3", absConvertCmd, inPath, outPath).Run(); err != nil {
		return err
	}

	// monsters bitmaps
	wgMonsters := sync.WaitGroup{}

	wg.Add(1)
	var monsterImages0 HashFile
	monsterImages0.Filename = "content/gfx/monsters/monsters0.d2p"
	monsterImages0.FriendlyName = "data/tmp/monster/monsters_0.d2p"

	wgMonsters.Add(1)
	go func() {
		defer wgMonsters.Done()
		DownloadHashImageFileInJson(files, monsterImages0)
	}()

	var monsterImages1 HashFile
	monsterImages1.Filename = "content/gfx/monsters/monsters0_1.d2p"
	monsterImages1.FriendlyName = "data/tmp/monster/monsters_1.d2p"

	wgMonsters.Add(1)
	go func() {
		defer wgMonsters.Done()
		DownloadHashImageFileInJson(files, monsterImages1)
	}()

	var monsterImages2 HashFile
	monsterImages2.Filename = "content/gfx/monsters/monsters0_2.d2p"
	monsterImages2.FriendlyName = "data/tmp/monster/monsters_2.d2p"

	wgMonsters.Add(1)
	go func() {
		defer wgMonsters.Done()
		DownloadHashImageFileInJson(files, monsterImages2)
	}()

	wgMonsters.Wait()

	inPath = fmt.Sprintf("%s/data/tmp/monster", path)
	outPath = fmt.Sprintf("%s/data/img/monster", path)
	absConvertCmd = fmt.Sprintf("%s/PyDofus/%s_unpack.py", path, "d2p")
	if err := exec.Command("/usr/local/bin/python3", absConvertCmd, inPath, outPath).Run(); err != nil {
		return err
	}

	// item vectors
	var itemImages0Vector HashFile
	itemImages0Vector.Filename = "content/gfx/items/vector0.d2p"
	itemImages0Vector.FriendlyName = "data/tmp/vector/vector_0.d2p"

	wgVectors := sync.WaitGroup{}

	wgVectors.Add(1)
	go func() {
		defer wgVectors.Done()
		if err := DownloadHashImageFileInJson(files, itemImages0Vector); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages1Vector HashFile
	itemImages1Vector.Filename = "content/gfx/items/vector0_1.d2p"
	itemImages1Vector.FriendlyName = "data/tmp/vector/vector_1.d2p"

	wgVectors.Add(1)
	go func() {
		defer wgVectors.Done()
		if err := DownloadHashImageFileInJson(files, itemImages1Vector); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages2Vector HashFile
	itemImages2Vector.Filename = "content/gfx/items/vector1.d2p"
	itemImages2Vector.FriendlyName = "data/tmp/vector/vector_2.d2p"

	wgVectors.Add(1)
	go func() {
		defer wgVectors.Done()
		if err := DownloadHashImageFileInJson(files, itemImages2Vector); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages3Vector HashFile
	itemImages3Vector.Filename = "content/gfx/items/vector1_1.d2p"
	itemImages3Vector.FriendlyName = "data/tmp/vector/vector_3.d2p"

	wgVectors.Add(1)
	go func() {
		defer wgVectors.Done()
		if err := DownloadHashImageFileInJson(files, itemImages3Vector); err != nil {
			log.Fatal(err)
		}
	}()

	var itemImages4Vector HashFile
	itemImages4Vector.Filename = "content/gfx/items/vector1_2.d2p"
	itemImages4Vector.FriendlyName = "data/tmp/vector/vector_4.d2p"

	wgVectors.Add(1)
	go func() {
		defer wgVectors.Done()
		if err := DownloadHashImageFileInJson(files, itemImages4Vector); err != nil {
			log.Fatal(err)
		}
	}()

	wgVectors.Wait()

	inPath = fmt.Sprintf("%s/data/tmp/vector", path)
	outPath = fmt.Sprintf("%s/data/vector/item", path)
	if err := exec.Command("/usr/local/bin/python3", absConvertCmd, inPath, outPath).Run(); err != nil {
		return err
	}

	return nil
}
