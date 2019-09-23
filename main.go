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
	datapack_name = kingpin.Flag("name", "Datapack name").Short('n').Default("random_loot").String()
	datapack_seed = kingpin.Flag("seed", "Datapack random seed").Short('s').Default(fmt.Sprintf("%d", time.Now().UnixNano())).Int64()
)

func main() {
	kingpin.Parse()

	// TODO: Make this user controlled
	r := rand.New(rand.NewSource(*datapack_seed))

	fileList := make([]string, 0)
	remaining := make([]string, 0)
	shuffled := make(map[string]string,)

	_ = filepath.Walk("loot_tables", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileList = append(fileList, path)
		remaining = append(remaining, path)

		return nil
	})

	for _, file := range fileList {
		i := r.Intn(len(remaining))
		shuffled[file] = remaining[i]
		log.Printf("Suffled: %s:%s", file, shuffled[file])
		remaining = remove(remaining, i)
	}

	createDataPack(fmt.Sprintf("%s_%d.zip", *datapack_name, *datapack_seed), shuffled)
	log.Printf("Wrote Datapack: %s_%d.zip", *datapack_name, *datapack_seed)
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
		if err := appendFile("data/minecraft/", strings.ReplaceAll(key, "\\", "/"), filename, zipw); err != nil {
			log.Fatalf("Failed to add file %s to zip: %s", filename, err)
		}
	}

	packMeta := map[string]interface{} {
		"pack": map[string]interface{} {
			"pack_format": 1,
			"description": "Test data pack",
		},
	}
	 if err := appendString("", "pack.mcmeta", interfaceToString(packMeta), zipw); err != nil {
	 	log.Fatal(err)
	 }

	loadMeta := map[string][]string {
		"values": {
			fmt.Sprintf("%s:reset", "test"),
		},
	}
	if err := appendString("data/minecraft/tags/functions/", "load.json", interfaceToString(loadMeta), zipw); err != nil {
		log.Fatal(err)
	}

	if err := appendString(fmt.Sprintf("data/%s/functions/", datapack_name), "reset.mcfunction", `tellraw @a ["",{"text":"Loot table randomizer by SethBling","color":"green"}]`, zipw); err != nil {
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

