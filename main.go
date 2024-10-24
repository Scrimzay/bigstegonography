package main

import (
	"bufio"
    "bytes"
    "flag"
    "fmt"
    "stenoprac/steg"
    "image"
    _ "image/png"
    "io/ioutil"
    //"log"
    "os"
    "strings"

	"github.com/auyer/steganography"
)


const encode = "encode"
const decode = "decode"

type sliceFlag []string

func (sf *sliceFlag) String() string {
    return strings.Join(*sf, " ")
}

func (sf *sliceFlag) Set(value string) error {
    *sf = append(*sf, value)
    return nil
}

var carrierFilesSlice sliceFlag
var carrierFiles = flag.String("carriers", "", "Carrier files in which the data is encoded (separated by space)")
var dataFile = flag.String("data", "", "Data file which is being encoded in the carrier")
var dataType = flag.String("data-type", "image", "Type of data being encoded (image/text)")
var resultFilesSlice sliceFlag
var resultFiles = flag.String("results", "", "Names of the result files (separated by space)")

func init() {
    flag.StringVar(carrierFiles, "c", "", "Carrier files in which the data is encoded (separated by space, shorthand for --carriers)")
    flag.Var(&carrierFilesSlice, "carrier", "Carrier file in which the data is encoded (can be used multiple times for multiple carriers)")
    flag.StringVar(dataFile, "d", "", "Data file which is being encoded in the carrier (shorthand for --data)")
    flag.Var(&resultFilesSlice, "result", "Name of the result file (can be used multiple times for multiple result file names)")
    flag.StringVar(resultFiles, "r", "", "Names of the result files (separated by space, shorthand for --results)")
    flag.StringVar(dataType, "t", "image", "Type of data being encoded (image/text), shorthand for --data-type")

    flag.Usage = func() {
        fmt.Fprintln(os.Stdout, "Usage: stegify [encode/decode] [flags...]")
        flag.PrintDefaults()
        fmt.Fprintln(os.Stdout, `NOTE: When multiple carriers are provided with different kinds of flags, the names provided through "carrier" flag are taken first and with "carriers"/"c" flags second. Same goes for the "result"/"results" flags.`)
        fmt.Fprintln(os.Stdout, `NOTE: When no results are provided, default values will be used for the names of the results.`)
    }
}

func main() {
    operation := parseOperation()
    flag.Parse()
    carriers := parseCarriers()
    results := parseResults()

    switch operation {
    case encode:
        if *dataType == "image" {
            // Encoding an image into images
            if len(results) == 0 { // if no results provided use defaults
                for i := range carriers {
                    results = append(results, fmt.Sprintf("result%d.png", i))
                }
            }
            if len(results) != len(carriers) {
                fmt.Fprintln(os.Stderr, "Carrier and result files count must be equal when encoding.")
                os.Exit(1)
            }
            if dataFile == nil || *dataFile == "" {
                fmt.Fprintln(os.Stderr, "Data file must be specified. Use --help for more information.")
                os.Exit(1)
            }
            err := steg.MultiCarrierEncodeByFileNames(carriers, *dataFile, results)
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
            }
        } else if *dataType == "text" {
            // Encoding text into an image
            if len(carriers) != 1 {
                fmt.Fprintln(os.Stderr, "Only one carrier image expected when encoding text.")
                os.Exit(1)
            }
            if len(results) == 0 {
                results = append(results, "encoded.png")
            } else if len(results) != 1 {
                fmt.Fprintln(os.Stderr, "Only one result file expected when encoding text.")
                os.Exit(1)
            }
            if dataFile == nil || *dataFile == "" {
                fmt.Fprintln(os.Stderr, "Data file must be specified for encoding text.")
                os.Exit(1)
            }
            // Read the message from dataFile
            message, err := ioutil.ReadFile(*dataFile)
            if err != nil {
                fmt.Fprintln(os.Stderr, "Error reading data file:", err)
                os.Exit(1)
            }
            // Open the carrier image
            img, err := OpenImageFromPath(carriers[0])
            if err != nil {
                fmt.Fprintln(os.Stderr, "Error opening carrier image:", err)
                os.Exit(1)
            }
            // Encode the message into the image
            encodedImg := new(bytes.Buffer)
            err = steganography.Encode(encodedImg, img, message)
            if err != nil {
                fmt.Fprintln(os.Stderr, "Error encoding message into image:", err)
                os.Exit(1)
            }
            // Write the encoded image to the result file
            err = ioutil.WriteFile(results[0], encodedImg.Bytes(), 0644)
            if err != nil {
                fmt.Fprintln(os.Stderr, "Error writing encoded image to file:", err)
                os.Exit(1)
            }
        } else {
            fmt.Fprintln(os.Stderr, "Unsupported data-type:", *dataType)
            os.Exit(1)
        }
    case decode:
        if *dataType == "image" {
            // Decoding an image from images
            if len(results) == 0 { // if no result provided use default
                results = append(results, "result.png")
            }
            if len(results) != 1 {
                fmt.Fprintln(os.Stderr, "Only one result file expected.")
                os.Exit(1)
            }
            err := steg.MultiCarrierDecodeByFileNames(carriers, results[0])
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
            }
        } else if *dataType == "text" {
            // Decoding text from an image
            if len(carriers) != 1 {
                fmt.Fprintln(os.Stderr, "Only one carrier image expected when decoding text.")
                os.Exit(1)
            }
            // Open the carrier image
            img, err := OpenImageFromPath(carriers[0])
            if err != nil {
                fmt.Fprintln(os.Stderr, "Error opening carrier image:", err)
                os.Exit(1)
            }
            // Get the message size
            sizeOfMessage := steganography.GetMessageSizeFromImage(img)
            // Decode the message
            msg := steganography.Decode(sizeOfMessage, img)
            // If results[0] is specified, write message to file, else print to stdout
            if len(results) == 0 || results[0] == "" {
                fmt.Print(string(msg))
            } else {
                err := ioutil.WriteFile(results[0], msg, 0644)
                if err != nil {
                    fmt.Fprintln(os.Stderr, "Error writing message to file:", err)
                    os.Exit(1)
                }
            }
        } else {
            fmt.Fprintln(os.Stderr, "Unsupported data-type:", *dataType)
            os.Exit(1)
        }
    default:
        fmt.Fprintln(os.Stderr, "Unsupported operation:", operation)
        os.Exit(1)
    }
}

func parseOperation() string {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Operation must be specified [encode/decode]. Use --help for more information.")
        os.Exit(1)
    }
    operation := os.Args[1]
    if operation != encode && operation != decode {
        helpFlags := map[string]bool{
            "--help": true,
            "-help":  true,
            "--h":    true,
            "-h":     true,
        }
        if helpFlags[operation] {
            flag.Parse()
            os.Exit(0)
        }
        fmt.Fprintf(os.Stderr, "Unsupported operation: %s. Only [encode/decode] operations are supported.\nUse --help for more information.", operation)
        os.Exit(1)
    }

    os.Args = append(os.Args[:1], os.Args[2:]...) // Remove the operation argument for flag parsing
    return operation
}

func parseCarriers() []string {
    carriers := make([]string, 0)
    if len(carrierFilesSlice) != 0 {
        carriers = append(carriers, carrierFilesSlice...)
    }

    if len(*carrierFiles) != 0 {
        carriers = append(carriers, strings.Split(*carrierFiles, " ")...)
    }

    if len(carriers) == 0 {
        fmt.Fprintln(os.Stderr, "Carrier file must be specified. Use --help for more information.")
        os.Exit(1)
    }

    return carriers
}

func parseResults() []string {
    results := make([]string, 0)
    if len(resultFilesSlice) != 0 {
        results = append(results, resultFilesSlice...)
    }

    if len(*resultFiles) != 0 {
        results = append(results, strings.Split(*resultFiles, " ")...)
    }

    return results
}

//returns an image.Image from a file path
func OpenImageFromPath(filename string) (image.Image, error) {
    inFile, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer inFile.Close()
    reader := bufio.NewReader(inFile)
    img, _, err := image.Decode(reader)
    if err != nil {
        return nil, err
    }
    return img, nil
}