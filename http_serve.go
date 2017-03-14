// Web interface for synchronizing UPR users databases
package tesis

import (
	"crypto/rsa"
	"crypto/tls"
	"github.com/dgrijalva/jwt-go"
	"html/template"
	"io/ioutil"
	"log"
	h "net/http"
	"path"
)

const AuthHd = "auth"

const (
	//Content files
	jquery = "jquery.js"
	//HTTPS server key files
	cert = "cert.pem"
	key  = "key.pem"
	//paths
	dashP = "/dash"
	syncP = "/sync"
)

var (
	notFound = []byte("Archivo no encontrado")
	notAuth  = []byte("No autenticado")
	tms      *template.Template
	fTms     = []string{"st/index.html", "st/dash.html"}
	pkey     *rsa.PrivateKey
	auth     Authenticator
	db       DBManager
	indexTm  *template.Template
	dashTm   *template.Template
)

// Creates a new instance of HTTPPortal and starts serving.
//  u: URL to serve
//  a: User authentication interface
//  q: Database manager interface
// The directory where the program is executed must have
// the following structure:
//  (. cert.pem key.pem
//    (st index.html index.js
//        dash.html dash.js
//        jquery.js util.js))
func ListenAndServe(u string, a Authenticator, d DBManager) {
	var bs []byte
	var e error
	auth, db = a, d
	// { auth,db:initialized }
	bs, e = ioutil.ReadFile(key)
	// { loaded.key.bs ≡ e = nil }
	if e == nil {
		// { loaded.key.bs }
		pkey, e = jwt.ParseRSAPrivateKeyFromPEM(bs)
		// { parsed.pkey ≡ e = nil }
		if e == nil {
			h.DefaultClient.Transport = &h.Transport{
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) h.RoundTripper),
			}
			// HTTP2 disabled
			e = parseTemplates()
			if e == nil {
				h.HandleFunc("/", indexH)
				h.HandleFunc("/s/", staticH)
				h.HandleFunc(dashP, dashH)
				h.HandleFunc(syncP, syncH)
				h.HandleFunc("/favicon.ico", h.NotFoundHandler().ServeHTTP)
				h.ListenAndServeTLS(u, cert, key, nil)
				// { serving tesis }
			}
		}
		// { serving tesis ≡ e = nil }
	}
	if e != nil {
		log.Print(e.Error())
	}
	return
}

// Handler of "/" path
func indexH(w h.ResponseWriter, r *h.Request) {
	if r.Method == h.MethodGet {
		tms.ExecuteTemplate(w, path.Base(fTms[0]), nil)
	}
}

// Handler of "/s" path
func staticH(w h.ResponseWriter, r *h.Request) {
	var file string
	file = path.Base(r.URL.Path)
	file = path.Join("st", file)
	h.ServeFile(w, r, file)
}

// exists.p ≡ ⟨∃ i: i ∈ `ls`: i = p⟩
func parseTemplates() (e error) {
	tms, e = template.ParseFiles(fTms...)
	// { e = nil ≡ exists.p ∧ parsed.tm }
	return
}

func dashH(w h.ResponseWriter, r *h.Request) {
	// { r.Method ∈ h.Method* }
	if r.Method == h.MethodPost {
		dashPost(w, r)
		// { written.UserName ∧ written.Cookie ≡ e = nil ∧ v
		//   ≢ written.(e.Error()) }
	} else if r.Method == h.MethodGet {
		dashGet(w, r)
		//
	}
}

func dashPost(w h.ResponseWriter, r *h.Request) {
	//globals
	// pkey: *rsa.PrivateKey,?
	// auth: Authenticator,?
	// AuthHd: string,?
	//end
	var e error
	var user, pass string
	var v bool
	e, v = r.ParseForm(), false
	if e == nil {
		user, pass = r.PostFormValue("user"), r.PostFormValue("pass")
		v = auth.Authenticate(user, pass)
	}
	// { v ≡ registered.user }
	if v {
		var u *User
		var t *jwt.Token
		var js string
		u = &User{UserName: user}
		t = jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), u)
		js, e = t.SignedString(pkey)
		// { signedString.js ≡ e = nil }
		if e == nil {
			var ck *h.Cookie
			ck = &h.Cookie{Name: AuthHd, Value: js}
			h.SetCookie(w, ck)
			writeInfo(w, u.UserName)
			// { written.(u.UserName) ∧ written.ck }
		}
	}

	if e != nil {
		w.Write([]byte(e.Error()))
	}
	// { written.(u.UserName) ∧ written.ck ≡ e = nil ∧ v
	//   ≢ written.(e.Error()) }
}

func dashGet(w h.ResponseWriter, r *h.Request) {
	//globals
	//p: *rsa.PublicKey,?
	//end
	var t *jwt.Token
	var e error
	t, e = parseToken(r, &pkey.PublicKey)
	// { e = nil ∧ t.Valid ≡ auth.(user.t) }
	if e == nil && t.Valid {
		var clm jwt.MapClaims
		var us string

		// { t.Claims is a jwt.MapClaims }
		clm = t.Claims.(jwt.MapClaims)
		us = clm["user"].(string)
		writeInfo(w, us)
		// { writtenInfo.us }
	} else {
		// { e ≠ nil ∨ ¬t.Valid}
		w.Write(notAuth)
		w.WriteHeader(401)
	}
	// { (writtenInfo.us ≢ written.notAuth }
}

func writeInfo(w h.ResponseWriter, user string) {
	// globals
	// db: DBManager,?
	// end
	var inf *Info
	var e error
	inf, e = db.UserInfo(user)
	// { loaded.inf ≡ e = nil }
	if e == nil {
		// { loaded.inf }
		tms.ExecuteTemplate(w, path.Base(fTms[1]), inf)
		// { written.inf }
	} else {
		// { ¬loaded.inf }
		w.Write([]byte(e.Error()))
		w.WriteHeader(500)
		// { written.(e.Error()) }
	}
	// { written.inf ≢ written.(e.Error()) ≡ e ≠ nil }
}

func parseToken(r *h.Request, p *rsa.PublicKey) (t *jwt.Token, e error) {
	var ck *h.Cookie
	ck, e = r.Cookie(AuthHd)
	// { readCookie.ck ≡ e = nil }
	if e == nil {
		t, e = jwt.Parse(ck.Value,
			func(x *jwt.Token) (a interface{}, d error) {
				a, d = p, nil
				return
			})
	}
	// { parsedToken.t ≡ e = nil }
	return
}

func syncH(w h.ResponseWriter, r *h.Request) {
	// { r.Method ∈ h.Method* }
	if r.Method == h.MethodPost {
		// { sync Info parsed }
		// { sync Info processed }
	}
}
