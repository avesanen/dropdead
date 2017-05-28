package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
)

type Dropdead struct {
	config *config
	db     *bolt.DB
	srv    *http.Server
	mux    *chi.Mux
}

func NewDropdead(conf *config) (dropdead *Dropdead, err error) {
	d := &Dropdead{
		config: conf,
	}

	d.mux = d.Mux()
	d.srv = &http.Server{Addr: d.config.Addr, Handler: d.mux}

	if _, err := os.Stat(d.config.UploadsPath); os.IsNotExist(err) {
		if err := os.Mkdir(d.config.UploadsPath, 0777); err != nil {
			return nil, err
		}
	}

	if _, err := os.Stat(d.config.UploadsPath + "/files"); os.IsNotExist(err) {
		if err := os.Mkdir(d.config.UploadsPath+"/files", 0777); err != nil {
			return nil, err
		}
	}

	// Check if dbpath exists, and create it if it doesn't.
	if _, err := os.Stat(d.config.DbPath); os.IsNotExist(err) {
		if err := os.Mkdir(d.config.DbPath, 0777); err != nil {
			return nil, err
		}
	}

	db, err := bolt.Open(d.config.DbPath+"/bolt.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	d.db = db

	err = d.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(GalleriesBucket)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Dropdead) Close() error {
	if err := d.db.Close(); err != nil {
		return err
	}
	return nil
}

func (d *Dropdead) ViewFileHandler(w http.ResponseWriter, req *http.Request) {
	galleryName := chi.URLParam(req, "galleryName")
	fileName := chi.URLParam(req, "fileName")

	g, err := d.LoadGallery(galleryName)
	if err != nil {
		log.Printf("Error loading gallery: %s", err.Error())
	}

	for _, f := range g.Files {
		if fileName == f.Name {
			http.ServeFile(w, req, d.config.UploadsPath+"/files/"+fileName)
			return
		}
	}
	d.ErrorHandler(w, req)
}

func (d *Dropdead) ViewGalleryHandler(w http.ResponseWriter, req *http.Request) {
	galleryName := chi.URLParam(req, "galleryName")

	g, err := d.LoadGallery(galleryName)
	if err != nil {
		log.Printf("Error loading gallery: %s", err.Error())
		d.ErrorHandler(w, req)
		return
	}
	if err := indexTemplate.ExecuteTemplate(w, "view_gallery", g); err != nil {
		log.Println(err.Error())
		d.ErrorHandler(w, req)
		return
	}
}

func (d *Dropdead) IndexHandler(w http.ResponseWriter, req *http.Request) {
	if err := indexTemplate.ExecuteTemplate(w, "index", nil); err != nil {
		log.Println(err.Error())
		d.ErrorHandler(w, req)
		return
	}
}

func (d *Dropdead) ErrorHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
	if err := indexTemplate.ExecuteTemplate(w, "error", nil); err != nil {
		log.Println(err.Error())
	}
}

type uploaderResponse struct {
	Status string `json:"status"`
	Url    string `json:"url"`
}

func (d *Dropdead) UploadHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := req.ParseMultipartForm(200000)
	if err != nil {
		log.Printf("Error uploading: %s", err.Error())
		fmt.Fprint(w, `{"status": "error"}`)
		return
	}

	formdata := req.MultipartForm
	files := formdata.File["files"]

	name := RandomName()
	g := &Gallery{
		Name: name,
	}

	filesUploaded := false
	for i, _ := range files {
		file, err := files[i].Open()
		if err != nil {
			log.Printf("Error opening request file: %s", err.Error())
			fmt.Fprint(w, `{"status": "error"}`)
			return
		}

		f := &File{}
		f.Name = RandomName() + filepath.Ext(files[i].Filename)
		f.Url = "/g/" + g.Name + "/" + f.Name

		ft := strings.SplitN(files[i].Header.Get("Content-Type"), "/", 2)
		if len(ft) != 2 {
			log.Println("Content-Type not in Type/Subtype format.")
			fmt.Fprint(w, `{"status": "error"}`)
			return
		}
		f.Type = ft[0]
		f.SubType = ft[1]

		// Crate file on disk
		out, err := os.Create(d.config.UploadsPath + "/files/" + f.Name)
		defer out.Close()
		if err != nil {
			log.Printf("Error creating file: %s", err.Error())
			fmt.Fprint(w, `{"status": "error"}`)
			return
		}

		// Copy uploaded file contents to the file
		_, err = io.Copy(out, file)
		if err != nil {
			log.Printf("Error copying file contents: %s", err.Error())
			fmt.Fprint(w, `{"status": "error"}`)
			return
		}
		g.Files = append(g.Files, f)
		filesUploaded = true
	}

	if filesUploaded {
		if err := d.SaveGallery(g); err != nil {
			log.Printf("Error saving gallery: %s", err.Error())
			fmt.Fprint(w, `{"status": "error"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status": "ok", "url":"/g/`+g.Name+`"}`)
	}
}

func (d *Dropdead) Mux() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.NotFound(d.ErrorHandler)
	r.Get("/", d.IndexHandler)
	r.Post("/upload", d.UploadHandler)
	r.Get("/g/:galleryName", d.ViewGalleryHandler)
	r.Get("/g/:galleryName/:fileName", d.ViewFileHandler)
	return r
}

func (d *Dropdead) ListenAndServe() (errChan chan error) {
	errChan = make(chan error)
	go func() {
		log.Printf("Dropdead starting at '%s'", d.config.Addr)
		if err := d.srv.ListenAndServe(); err != nil {
			errChan <- err
		}
		d.srv.Close()
	}()
	return errChan
}

func (d *Dropdead) Shutdown() error {
	log.Printf("Dropdead shutting down")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return d.srv.Shutdown(ctx)
}
