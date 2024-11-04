package main

import (
	"bufio"
    "bytes"
    "fmt"
    "stenoprac/steg"
    "image"
    _ "image/png"
    "io/ioutil"
    "os"
    "net/http"
    "path/filepath"
    "io"
    "archive/zip"

	"github.com/auyer/steganography"
    "github.com/gin-gonic/gin"
)


func main() {
    r := gin.Default()
    r.LoadHTMLFiles("index.html") // Load your HTML file

    // Ensure uploads directory exists
    if _, err := os.Stat("uploads"); os.IsNotExist(err) {
        os.Mkdir("uploads", os.ModePerm)
    }

    // Serve the HTML form
    r.GET("/", func(c *gin.Context) {
        c.HTML(http.StatusOK, "index.html", nil)
    })

    // Handle encoding requests
    r.POST("/encode", func(c *gin.Context) {
        fmt.Println("Encode handler invoked")
        stegoType := c.PostForm("stegoType")
        operation := "encode"

        if stegoType == "image" {
            // Handle image inside image encoding
            handleImageInImage(c, operation)
        } else if stegoType == "text" {
            // Handle text inside image encoding
            handleTextInImage(c, operation)
        } else {
            c.String(http.StatusBadRequest, "Invalid steganography type")
        }
    })

    // Handle decoding requests
    r.POST("/decode", func(c *gin.Context) {
        fmt.Println("Decode handler invoked")
        stegoType := c.PostForm("stegoType")
        operation := "decode"

        if stegoType == "image" {
            // Handle image inside image decoding
            handleImageInImage(c, operation)
        } else if stegoType == "text" {
            // Handle text inside image decoding
            handleTextInImage(c, operation)
        } else {
            c.String(http.StatusBadRequest, "Invalid steganography type")
        }
    })

    fmt.Println("Server started at http://localhost:8080")
    r.Run(":8080")
}

// Handler for image inside image encoding/decoding
func handleImageInImage(c *gin.Context, operation string) {
    if operation == "encode" {
        // Get hidden image
        hiddenImageFile, err := c.FormFile("hiddenImage")
        if err != nil {
            c.String(http.StatusBadRequest, "Error retrieving the hidden image: %v", err)
            return
        }
        hiddenImagePath := filepath.Join("uploads", hiddenImageFile.Filename)
        err = c.SaveUploadedFile(hiddenImageFile, hiddenImagePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error saving hidden image: %v", err)
            return
        }

        // Get cover images
        coverImages := c.Request.MultipartForm.File["coverImages"]
        if len(coverImages) == 0 {
            c.String(http.StatusBadRequest, "At least one cover image is required")
            return
        }

        carriers := make([]string, 0)
        for _, fileHeader := range coverImages {
            coverImagePath := filepath.Join("uploads", fileHeader.Filename)
            err = c.SaveUploadedFile(fileHeader, coverImagePath)
            if err != nil {
                c.String(http.StatusInternalServerError, "Error saving cover image: %v", err)
                return
            }
            carriers = append(carriers, coverImagePath)
        }

        // Generate result file names
        results := make([]string, len(carriers))
        for i := range carriers {
            results[i] = fmt.Sprintf("result%d.png", i)
        }

        // Perform encoding
        err = steg.MultiCarrierEncodeByFileNames(carriers, hiddenImagePath, results)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error encoding image: %v", err)
            return
        }

        // Send the encoded images as a ZIP file
        zipPath := "encoded_images.zip"
        err = createZip(results, zipPath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error creating ZIP file: %v", err)
            return
        }

        c.Header("Content-Disposition", "attachment; filename=encoded_images.zip")
        c.Header("Content-Type", "application/zip")
        c.File(zipPath)

        // Clean up uploaded and generated files
        cleanupFiles(append(carriers, hiddenImagePath), append(results, zipPath))

    } else if operation == "decode" {
        // Get carrier images
        carrierImages := c.Request.MultipartForm.File["carrierImages"]
        if len(carrierImages) == 0 {
            c.String(http.StatusBadRequest, "At least one carrier image is required")
            return
        }

        carriers := make([]string, 0)
        for _, fileHeader := range carrierImages {
            carrierImagePath := filepath.Join("uploads", fileHeader.Filename)
            err := c.SaveUploadedFile(fileHeader, carrierImagePath)
            if err != nil {
                c.String(http.StatusInternalServerError, "Error saving carrier image: %v", err)
                return
            }
            carriers = append(carriers, carrierImagePath)
        }

        // Result file name
        resultFile := "decoded_image.png"

        // Perform decoding
        err := steg.MultiCarrierDecodeByFileNames(carriers, resultFile)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error decoding image: %v", err)
            return
        }

        c.FileAttachment(resultFile, resultFile)

        // Clean up uploaded and generated files
        cleanupFiles(carriers, []string{resultFile})
    }
}

// Handler for text inside image encoding/decoding
func handleTextInImage(c *gin.Context, operation string) {
    if operation == "encode" {
        // Get text file
        textFileHeader, err := c.FormFile("textFile")
        if err != nil {
            c.String(http.StatusBadRequest, "Error retrieving the text file: %v", err)
            return
        }
        textFilePath := filepath.Join("uploads", textFileHeader.Filename)
        err = c.SaveUploadedFile(textFileHeader, textFilePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error saving text file: %v", err)
            return
        }

        // Get cover image
        coverImageHeader, err := c.FormFile("coverImage1")
        if err != nil {
            c.String(http.StatusBadRequest, "Error retrieving the cover image: %v", err)
            return
        }
        coverImagePath := filepath.Join("uploads", coverImageHeader.Filename)
        err = c.SaveUploadedFile(coverImageHeader, coverImagePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error saving cover image: %v", err)
            return
        }

        // Read the message from text file
        message, err := ioutil.ReadFile(textFilePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error reading text file: %v", err)
            return
        }

        // Open the carrier image
        img, err := OpenImageFromPath(coverImagePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error opening carrier image: %v", err)
            return
        }

        // Encode the message into the image
        encodedImg := new(bytes.Buffer)
        err = steganography.Encode(encodedImg, img, message)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error encoding message into image: %v", err)
            return
        }

        // Write the encoded image to a file
        resultFile := "encoded.png"
        err = ioutil.WriteFile(resultFile, encodedImg.Bytes(), 0644)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error writing encoded image to file: %v", err)
            return
        }

        c.FileAttachment(resultFile, resultFile)

        // Clean up uploaded and generated files
        cleanupFiles([]string{textFilePath, coverImagePath}, []string{resultFile})

    } else if operation == "decode" {
        // Get carrier image
        carrierImageHeader, err := c.FormFile("carrierImage")
        if err != nil {
            c.String(http.StatusBadRequest, "Error retrieving the carrier image: %v", err)
            return
        }
        carrierImagePath := filepath.Join("uploads", carrierImageHeader.Filename)
        err = c.SaveUploadedFile(carrierImageHeader, carrierImagePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error saving carrier image: %v", err)
            return
        }

        // Open the carrier image
        img, err := OpenImageFromPath(carrierImagePath)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error opening carrier image: %v", err)
            return
        }

        // Get the message size
        sizeOfMessage := steganography.GetMessageSizeFromImage(img)
        // Decode the message
        msg := steganography.Decode(sizeOfMessage, img)

        // Send the message as a text file
        resultFile := "message.txt"
        err = ioutil.WriteFile(resultFile, msg, 0644)
        if err != nil {
            c.String(http.StatusInternalServerError, "Error writing message to file: %v", err)
            return
        }

        c.FileAttachment(resultFile, resultFile)

        // Clean up uploaded and generated files
        cleanupFiles([]string{carrierImagePath}, []string{resultFile})
    }
}

// Utility function to open an image from a file path
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

// createZip creates a ZIP file at zipPath containing the files listed in the files slice.
func createZip(files []string, zipPath string) error {
    // Create the ZIP file
    zipFile, err := os.Create(zipPath)
    if err != nil {
        return err
    }
    defer zipFile.Close()

    // Create a new zip archive.
    zipWriter := zip.NewWriter(zipFile)
    defer zipWriter.Close()

    // Add files to the ZIP archive
    for _, file := range files {
        err := addFileToZip(zipWriter, file)
        if err != nil {
            return err
        }
    }

    return nil
}

// addFileToZip adds an individual file to the ZIP archive.
func addFileToZip(zipWriter *zip.Writer, filename string) error {
    // Open the file to be added to the ZIP
    fileToZip, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer fileToZip.Close()

    // Get file information to set the correct header in the ZIP file
    info, err := fileToZip.Stat()
    if err != nil {
        return err
    }

    // Create a zip header from the file info
    header, err := zip.FileInfoHeader(info)
    if err != nil {
        return err
    }

    // Use the base name of the file (without the full path)
    header.Name = filepath.Base(filename)

    // Set the compression method
    header.Method = zip.Deflate

    // Create a writer for the file in the ZIP archive
    writer, err := zipWriter.CreateHeader(header)
    if err != nil {
        return err
    }

    // Copy the file content into the ZIP archive
    _, err = io.Copy(writer, fileToZip)
    if err != nil {
        return err
    }

    return nil
}

// Utility function to clean up files
func cleanupFiles(uploadedFiles []string, generatedFiles []string) {
    for _, file := range uploadedFiles {
        os.Remove(file)
    }
    for _, file := range generatedFiles {
        os.Remove(file)
    }
}