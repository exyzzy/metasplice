package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

func main() {
	srcPtr := flag.String("src", "", "source file")
	destPtr := flag.String("dest", "", "destination file")
	modePtr := flag.String("mode", "splice", "mode: splice (default), extract, generator")

	flag.Parse()
	if flag.NFlag() == 0 {
		fmt.Println(" valid usage is:")
		//splice: default applies diff file to current source
		fmt.Println("  metasplice -src=sourcefile.diff.go -dest=destinationfile.go")
		fmt.Println("  metasplice -src=sourcefile.diff.go -dest=destinationfile.go -mode=splice")
		//extract: create a new .diff.go from the splice points in the source
		fmt.Println("  metasplice -src=sourcefile.go -dest=destinationfile.diff.go -mode=extract")
		//applyfile: create the applysplice.go file from the extractsplice.go file
		fmt.Println("  metasplice -src=extractsplice.go -dest=applysplice.go -mode=applyfile")
		os.Exit(1)
	}
	if (filepath.Base(*srcPtr) == "") || (filepath.Base(*destPtr) == "") {
		log.Panic("Invalid File Params")
	}
	fmt.Print("metasplice -src=", *srcPtr, " -dest=", *destPtr)
	if *modePtr == "extract" {
		fmt.Println(" -mode=extract")
		//extract splices
		err := extractFile(*srcPtr, *destPtr)
		if err != nil {
			log.Panic(err.Error())
		}
	} else if *modePtr == "applyfile" {
		fmt.Println(" -mode=applyfile")
		//make generate file
		err := applyFile(*srcPtr, *destPtr)
		if err != nil {
			log.Panic(err.Error())
		}
	} else {
		fmt.Println()
		//apply splices
		err := spliceFile(*srcPtr, *destPtr)
		if err != nil {
			log.Panic(err.Error())
		}
	}
}

//go through src, find all text between #SPLICE# tag ... #SPLICE# end and create template file
func extractFile(src, dest string) error {
	//get file type
	ext := strings.TrimSpace(strings.ToLower(filepath.Ext(src)))
	dat, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	destPath := filepath.Dir(dest)
	err = os.MkdirAll(destPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	fout, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fout.Close()

	//find splice tag for .go/.js and .html "\\#SPLICE# foo ", or "<!--#SPLICE# foo -->"
	re1 := regexp.MustCompile(`(\/\/\s*#SPLICE#\s+\w+.*|<!--\s*#SPLICE#\s+\w+\s+-->)`)
	//find the tagname within a splice tag
	re2 := regexp.MustCompile(`#SPLICE#\s+(\w+)`)

	//FindAllIndex returns array of pair: [index of string start and index 1 past end]
	idx := re1.FindAllIndex(dat, -1)
	_, err = fout.WriteString("package main\n")
	if err != nil {
		return err
	}
	splicestart := 0
	splicename := "end"
	for _, v := range idx { //go through each pair of FindAllIndex
		subdat := dat[v[0]:v[1]]

		loc := re2.FindSubmatchIndex(subdat)
		if (len(loc) == 0) || (loc[3]-loc[2] == 0) {
			err = errors.New("No tagname detected after #SPLICE#")
			return err
		}
		tagname := string(subdat[loc[2]:loc[3]]) //regex submatch
		if tagname == "end" {                    //if tagname is "end" write the template splice tag name & splice code
			if splicename == "end" {
				err = errors.New("No beginning splice name detected")
				return err
			}
			_, err = fout.WriteString("{[< define \"" + splicename + "\" >]}\n")
			if err != nil {
				return err
			}
			if ext == ".html" { //if html then cap off the multiline comment
				_, err = fout.WriteString("-->\n")
				if err != nil {
					return err
				}
			}
			_, err = fout.WriteString(string(dat[splicestart:v[0]]) + "\n") //up to beginning of end splice
			if err != nil {
				return err
			}

			if ext == ".html" { //cap on both sides
				_, err = fout.WriteString("\n<!--")
				if err != nil {
					return err
				}
			}
			_, err = fout.WriteString("{[< end >]}\n")
			if err != nil {
				return err
			}
		} else { //otherwise capture the splice code start and tagname
			splicestart = v[1] //start after tag
			splicename = tagname
		}
	}
	fout.Sync() //all done
	return err
}

//create the applysplice.go file from the extractsplice.go file
func applyFile(src, dest string) error {
	dat, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	destPath := filepath.Dir(dest)
	err = os.MkdirAll(destPath, os.FileMode(0755))
	if err != nil {
		return err
	}

	fout, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fout.Close()

	re := regexp.MustCompile(`go:generate.+\n`)
	idx := re.FindAllIndex(dat, -1)
	destName := ""
	srcName := ProjName()
	_, err = fout.WriteString("package splice\n\n")
	if err != nil {
		return err
	}
	for _, v := range idx {
		//todo: switch to regex instead of fields
		fields := strings.Fields(string(dat[v[0]:v[1]]))
		switch fields[1] {
		case "mkdir":
			if destName == "" {
				destName = DestName(fields[len(fields)-1])
			}
			str := "//go:generate mkdir -p "
			str += strings.Replace(fields[len(fields)-1], destName, srcName, 1) + "\n"
			_, err = fout.WriteString(str)
			if err != nil {
				return err
			}
		case "cp":
			str := "//go:generate cp "
			if fields[len(fields)-3] != "cp" { //preserve a single cp flag if exists (like -R)
				str += fields[len(fields)-3] + " "
			}
			str += fields[len(fields)-2] + " "
			str += strings.Replace(fields[len(fields)-1], destName, srcName, 1) + "\n"
			_, err = fout.WriteString(str)
			if err != nil {
				return err
			}
		case "metasplice":
			if fields[len(fields)-1] == "-mode=applyfile" {
				break
			}
			str := "//go:generate metasplice "
			li := strings.LastIndex(fields[2], ".")
			str += fields[2][:li] + ".diff" + fields[2][li:] + " "
			s := strings.Replace(fields[3], destName, srcName, 1)
			str += strings.Replace(s, ".diff", "", 1) + "\n"
			_, err = fout.WriteString(str)
			if err != nil {
				return err
			}
		}
	}
	fout.Sync() //all done
	return err
}

//execute the nested template splice file against the commented template invocation in dest, specific delims
func spliceFile(src, dest string) error {
	destb, err := ioutil.ReadFile(dest)
	if err != nil {
		return err
	}

	tt := template.Must(template.New("dest").Delims("{[<", ">]}").Parse(string(destb)))
	template.Must(tt.New("src").Delims("{[<", ">]}").ParseFiles(src))

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	err = tt.ExecuteTemplate(file, "dest", nil)
	file.Close()
	return err
}

//assume that src path always ends in /extractsplice under project
func ProjName() string {
	wd, _ := os.Getwd()
	li := strings.LastIndex(wd, "/extractsplice")
	fi := strings.LastIndex(wd[:li], "/")
	return wd[fi+1 : li]
}

//asssume a mkdir that ends in /destname as first line
func DestName(path string) string {
	li := strings.LastIndex(path, "/")
	return (path[li+1:])
}
