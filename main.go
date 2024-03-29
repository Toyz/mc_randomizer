package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	datapack_name = "random_loot"
	datapack_seed = kingpin.Flag("seed", "Datapack random seed").Short('s').Default(fmt.Sprintf("%d", time.Now().UnixNano())).Int64()
)

func main() {
	kingpin.Parse()
	r := rand.New(rand.NewSource(*datapack_seed))
	remaining := make([]string, 0)
	shuffle := make(map[string]string, )

	datapack_name = fmt.Sprintf("%s_%d", datapack_name, *datapack_seed)

	lootTableFiles := make([]string, 0)
	_ = filepath.Walk("loot_tables", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		lootTableFiles = append(lootTableFiles, path)
		remaining = append(remaining, path)

		return nil
	})

	for _, file := range lootTableFiles {
		i := r.Intn(len(remaining))
		shuffle[file] = remaining[i]
		log.Printf("Suffled: %s", file)
		remaining = remove(remaining, i)
	}

	createDataPack(fmt.Sprintf("%s.zip", datapack_name), shuffle)
	log.Printf("Wrote Datapack: %s.zip", datapack_name)
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func createDataPack(name string, files map[string]string) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	file, err := os.OpenFile(name, flags, 0644)
	if err != nil {
		log.Fatalf("Failed to open zip for writing: %s", err)
	}
	defer file.Close()

	zipw := zip.NewWriter(file)
	defer zipw.Close()

	for key, filename := range files {
		log.Printf("Wrote: %s", key)
		if err := appendFile("data/minecraft/", strings.ReplaceAll(key, "\\", "/"), filename, zipw); err != nil {
			log.Fatalf("Failed to add file %s to zip: %s", filename, err)
		}
	}

	packMeta := map[string]interface{}{
		"pack": map[string]interface{}{
			"pack_format": 1,
			"description": "Test data pack",
		},
	}
	if err := appendString("", "pack.mcmeta", interfaceToString(packMeta), zipw); err != nil {
		log.Fatal(err)
	}

	loadMeta := map[string][]string{
		"values": {
			fmt.Sprintf("%s:reset", datapack_name),
			fmt.Sprintf("%s:seed", datapack_name),
		},
	}
	if err := appendString("data/minecraft/tags/functions/", "load.json", interfaceToString(loadMeta), zipw); err != nil {
		log.Fatal(err)
	}

	if err := appendString(fmt.Sprintf("data/%s/functions/", datapack_name), "reset.mcfunction", `tellraw @a ["",{"text":"Loot table randomizer by SethBling","color":"green"}]`, zipw); err != nil {
		log.Fatal(err)
	}

	if err := appendString(fmt.Sprintf("data/%s/functions/", datapack_name), "seed.mcfunction", fmt.Sprintf(`tellraw @a ["",{"text":"Randomizer Seed: %d","color":"green"}]`, *datapack_seed), zipw); err != nil {
		log.Fatal(err)
	}

	loadFunctions := `
	function %s:reset
	function %s:seed
	`

	if err := appendString("data/minecraft/functions/", "load.mcfunction", fmt.Sprintf(loadFunctions, datapack_name, datapack_name), zipw); err != nil {
		log.Fatal(err)
	}
}

func appendFile(path, filename, actual string, zipw *zip.Writer) error {
	file, err := os.Open(actual)
	if err != nil {
		return fmt.Errorf("Failed to open %s: %s", filename, err)
	}
	defer file.Close()

	wr, err := zipw.Create(path + filename)
	if err != nil {
		msg := "Failed to create entry for %s in zip file: %s"
		return fmt.Errorf(msg, filename, err)
	}

	if _, err := io.Copy(wr, file); err != nil {
		return fmt.Errorf("Failed to write %s to zip: %s", filename, err)
	}

	return nil
}

func appendString(path, filename, data string, zipw *zip.Writer) error {
	wr, err := zipw.Create(path + filename)
	if err != nil {
		msg := "Failed to create entry for %s in zip file: %s"
		return fmt.Errorf(msg, filename, err)
	}

	if _, err := io.Copy(wr, strings.NewReader(data)); err != nil {
		return fmt.Errorf("Failed to write %s to zip: %s", filename, err)
	}

	return nil
}

func interfaceToString(in interface{}) string {
	data, _ := json.Marshal(in)

	return string(data)
}
