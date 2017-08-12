package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/axgle/mahonia"
)

type testQuestion struct {
	id        string
	question  string
	testType  string
	classType string
	options   []string
	answer    string
}

var exams = make(chan string)
var tests = make(chan testQuestion)

func get(url string) {
	exams <- httpGet(url)
}

func httpGet(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
	}
	dec := mahonia.NewDecoder("gbk")
	str := dec.ConvertString(string(body))
	return str
}

func main() {
	config := [][]int{
		{1490, 26}, //tikubiaohao, the num of pages
		{1471, 55},
	}
	pagesUrls := []string{}
	for _, v := range config {
		for i := 1; i <= v[1]; i++ {
			pagesUrls = append(pagesUrls,
				"http://safeexam.hdu.edu.cn/redir.php?catalog_id=6&cmd=learning&tikubh="+strconv.Itoa(v[0])+"&page="+strconv.Itoa(i))
		}
	}
	for _, page := range pagesUrls {
		fmt.Fprintln(os.Stderr, "get ", page)
		go get(page)
	}
	go parseTest()
	for t := range tests {
		if t.testType == "判断" {
			fmt.Println("INSERT INTO JExam (ID,problem,answer) VALUES (" + t.id + ",'" + t.question + "','" + t.answer + "');")
		} else {
			str := "INSERT INTO CExam (ID,problem,choice_A,choice_B,choice_C,choice_D,answer) VALUES (" + t.id + ",'" + t.question + "'"
			for _, op := range t.options {
				str += ", '" + op + "'"
			}
			if len(t.options) == 3 {
				str += ", '空'"
			}
			str += ", '" + t.answer + "');"
			fmt.Println(str)
		}
	}

}

func parseTest() {
	var classType, testType string
	titleRe := regexp.MustCompile("<title>(.+)安全题（(.+)） - 高校实验室安全考试系统</title>")
	testRe := regexp.MustCompile(`<div class="shiti"><h3>([0-9]*)、(.*)</h3><ul class="xuanxiang_[^"]*">(.*)</ul></div> <span [^>]*>（标准答案：\s*(.*)\s*）</span>`)
	optionRe := regexp.MustCompile(`(?U)<li><input[^>]*><label [^>]*>(.*)</label></li>`)
	for exam := range exams {
		res := titleRe.FindStringSubmatch(exam)
		classType, testType = res[1], res[2]
		testRes := testRe.FindAllStringSubmatch(exam, -1)
		for _, test := range testRes {
			var t testQuestion
			t.testType = strings.TrimSpace(testType)
			t.classType = strings.TrimSpace(classType)
			t.id, t.question, t.answer = strings.TrimSpace(test[1]), strings.TrimSpace(test[2]), strings.TrimSpace(test[4])
			opRes := optionRe.FindAllStringSubmatch(test[3], -1)
			for _, op := range opRes {
				t.options = append(t.options, strings.TrimSpace(op[1]))
			}
			tests <- t
		}
	}
	close(tests)
}
