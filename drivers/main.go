package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Output struct {
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata"`
	Error    string         `json:"error,omitempty"`
}

func main() {
	filePath := flag.String("file", "", "Path to the file to process")
	flag.Parse()

	if *filePath == "" {
		sendError("No file provided")
		return
	}

	// 模拟解压缩并提取元数据
	r, err := zip.OpenReader(*filePath)
	if err != nil {
		// 如果不是 ZIP，也返回成功但元数据为空，或者根据业务报错
		sendError(fmt.Sprintf("Failed to open zip: %v", err))
		return
	}
	defer r.Close()

	fileList := []string{}
	hasScenarioConfig := false
	for _, f := range r.File {
		fileList = append(fileList, f.Name)
		if filepath.Base(f.Name) == "scenario.json" {
			hasScenarioConfig = true
		}
	}

	// 构造并输出结果
	out := Output{
		Status: "success",
		Metadata: map[string]any{
			"files_count": len(fileList),
			"has_config":  hasScenarioConfig,
			"driver":      "generic-zip-inspector-v1",
		},
	}

	// 模拟针对想定的特定提取（业务逻辑落在这里，而不是后端核心代码）
	if hasScenarioConfig {
		out.Metadata["scenario_type"] = "standard_mission"
		out.Metadata["estimated_duration"] = 3600 // 模拟解析出的时长
	}

	data, _ := json.Marshal(out)
	fmt.Println(string(data))
}

func sendError(msg string) {
	out := Output{
		Status: "failed",
		Error:  msg,
	}
	data, _ := json.Marshal(out)
	fmt.Println(string(data))
	os.Exit(0) // 驱动本身运行成功，只是业务层面失败
}
