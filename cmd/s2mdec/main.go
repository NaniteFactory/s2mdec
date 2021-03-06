package main

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/icza/s2prot"
	"github.com/sc2-arcade-watcher/s2mdec"
)

var args []string // non-flag args
var bFlagCompact bool
var bFlagUnlabeled bool

func init() {
	flag.BoolVar(&bFlagCompact, "c", false, "Compact: print out json without indentations")
	flag.BoolVar(&bFlagUnlabeled, "u", false, "Unlabeled: print out json labeled with numbers instead of each field's respective name (applies only to s2mi and s2mh files)")
	flag.Parse()
	args = flag.Args()
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("fatal error:", r)
		}
	}()
	//
	// len args
	switch len(args) {
	case 1: // decode a single file
		// fileIn
		fileIn, err := os.Open(args[0])
		if err != nil {
			log.Println(err)
			return
		}
		defer fileIn.Close()
		// dataIn
		dataIn, err := ioutil.ReadAll(fileIn)
		if err != nil {
			log.Println(err)
			return
		}
		{ // switch ext
			ext := strings.ToLower(filepath.Ext(fileIn.Name()))
			switch ext {
			case ".s2mi":
				// unlabeled
				unlabeled, ok := s2mdec.NewVersionedDec(dataIn).ReadStruct().(s2prot.Struct)
				if !ok {
					log.Println("invalid s2mi")
					return
				}
				// bFlagUnlabeled
				var output s2prot.Struct
				if bFlagUnlabeled {
					output = unlabeled
				} else {
					labeled, err := s2mdec.ReadS2MI(unlabeled)
					if err != nil {
						log.Println("s2mi:", err)
						return
					}
					output = labeled
				}
				// bFlagCompact
				if errJSON := writeJSON(os.Stdout, output, !bFlagCompact); errJSON != nil {
					log.Println("s2mi:", errJSON)
					return
				}
				return
			case ".s2mh":
				// unlabeled
				unlabeled, ok := s2mdec.NewVersionedDec(dataIn).ReadStruct().(s2prot.Struct)
				if !ok {
					log.Println("invalid s2mh")
					return
				}
				// bFlagUnlabeled
				var output s2prot.Struct
				if bFlagUnlabeled {
					output = unlabeled
				} else {
					labeled, err := s2mdec.ReadS2MH(unlabeled)
					if err != nil {
						log.Println("s2mh:", err)
						return
					}
					output = labeled
				}
				// bFlagCompact
				if errJSON := writeJSON(os.Stdout, output, !bFlagCompact); errJSON != nil {
					log.Println("s2mh:", errJSON)
					return
				}
				return
			case ".s2ml":
				// translation
				translation, err := s2mdec.ReadS2ML(dataIn)
				if err != nil {
					log.Println("s2ml:", err)
					return
				}
				// bFlagCompact
				if errJSON := writeJSON(os.Stdout, translation, !bFlagCompact); errJSON != nil {
					log.Println("s2ml:", errJSON)
					return
				}
				return
			case ".s2gs":
				// zlib
				rZlib, errZlib := zlib.NewReader(bytes.NewReader(dataIn[16:]))
				if errZlib != nil {
					log.Println("s2gs:", errZlib)
					return
				}
				defer rZlib.Close()
				// dataIn2
				dataIn2, errDataIn2 := ioutil.ReadAll(rZlib)
				if errDataIn2 != nil {
					log.Println("s2gs:", errDataIn2)
					return
				}
				// unlabeled
				unlabeled, ok := s2mdec.NewVersionedDec(dataIn2).ReadStruct().(s2prot.Struct)
				if !ok {
					log.Println("invalid s2gs")
					return
				}
				// bFlagUnlabeled
				var output s2prot.Struct
				if bFlagUnlabeled {
					output = unlabeled
				} else {
					output = unlabeled // not supported yet
				}
				// bFlagCompact
				if errJSON := writeJSON(os.Stdout, output, !bFlagCompact); errJSON != nil {
					log.Println("s2gs:", errJSON)
					return
				}
				return
			default:
				// unlabeled
				unlabeled, ok := s2mdec.NewVersionedDec(dataIn).ReadStruct().(s2prot.Struct)
				if !ok {
					log.Println("Unsupported file extension:", ext)
					return
				}
				// bFlagCompact
				if errJSON := writeJSON(os.Stdout, unlabeled, !bFlagCompact); errJSON != nil {
					log.Println("Unsupported file extension:", ext)
					return
				}
				return
			}
		}
	case 2: // merging two files s2mh and s2ml
		const nFiles = 2
		fileIn := make([]*os.File, nFiles)
		dataIn := make([][]byte, nFiles)
		for i := 0; i < nFiles; i++ {
			var err error
			// fileIn
			fileIn[i], err = os.Open(args[i])
			if err != nil {
				log.Println(err)
				return
			}
			defer fileIn[i].Close()
			// dataIn
			dataIn[i], err = ioutil.ReadAll(fileIn[i])
			if err != nil {
				log.Println(err)
				return
			}
		}
		// prepare
		s2mh, s2ml := s2prot.Struct(nil), s2mdec.MapLocale(nil)
		{ // switch ext s2mh
			ext := strings.ToLower(filepath.Ext(fileIn[0].Name()))
			switch ext {
			case ".s2mh":
				unlabeled, ok := s2mdec.NewVersionedDec(dataIn[0]).ReadStruct().(s2prot.Struct)
				if !ok {
					log.Println("invalid s2mh")
					return
				}
				var err error
				s2mh, err = s2mdec.ReadS2MH(unlabeled)
				if err != nil {
					log.Println("s2mh:", err)
					return
				}
			default:
				log.Println("Unsupported file extension:", ext)
				return
			}
		}
		{ // switch ext s2ml
			ext := strings.ToLower(filepath.Ext(fileIn[1].Name()))
			switch ext {
			case ".s2ml":
				var err error
				s2ml, err = s2mdec.ReadS2ML(dataIn[1])
				if err != nil {
					log.Println("s2ml:", err)
					return
				}
			default:
				log.Println("Unsupported file extension:", ext)
				return
			}
		}
		// merged
		merged, err := s2mdec.S2MHApplyS2ML(s2mh, s2ml, nil)
		if err != nil {
			log.Println("s2mh plus s2ml:", err)
			return
		}
		// bFlagCompact
		if errJSON := writeJSON(os.Stdout, merged, !bFlagCompact); errJSON != nil {
			log.Println("s2mh plus s2ml:", errJSON)
			return
		}
		return
	default: // unexpected len(args)
		log.Println("Invalid argument.")
		return
	}
}

func writeJSON(w io.Writer, v interface{}, indent bool) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	} else {
		enc.SetIndent("", "")
	}
	return enc.Encode(v)
}
