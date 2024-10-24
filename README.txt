its just 2 different packages merged together into one big package that does everything
just wanted to see if i can take text hidden to image code and image hidden to image code and merge them into one big thing that does both
not trying to actually steal code and claim it as my own
i took the example code from auyer/stegonography and 
pretty much just copy pasted the DimitarPetrov/stegify repo minus the examples folder (im a code stealer Aware)

heres some examples take it as you would

Encode text into an image:
go run main.go encode --data message.txt --data-type text --carrier stegosaurus.png --result encoded_text_image_merge1.png

Decode text from an image:
go run main.go decode --data-type text --carrier encoded_text_image_merge1.png --result decoded_text_from_merge.txt

Encode an image into an image:
go run main.go encode --data 61meskq+zGL.jpg --carrier stegosaurus.jpg --result encoded_image_image_merge1.jpg

Decode an image from an image:
go run main.go decode --carrier encoded_image_image_merge1.jpg --result decoded_image_image_merge1.jpg