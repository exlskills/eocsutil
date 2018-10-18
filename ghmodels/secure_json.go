package ghmodels

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/exlinc/golang-utils/jsonhttp"
	"net/http"
	"reflect"
	"strings"
)

func GHSecureJSONDecodeAndCatchForAPI(w http.ResponseWriter, r *http.Request, hubSecret string, outStruct interface{}) error {
	buf := bytes.Buffer{}
	buf.ReadFrom(r.Body)
	reqBody := buf.Bytes()
	icSig := r.Header.Get("X-Hub-Signature")
	if icSig == "" || !strings.HasPrefix(icSig, "sha1=") {
		jsonhttp.JSONForbiddenError(w, "Missing request signature", "")
		return errors.New("missing signature")
	}
	icSig = strings.Replace(icSig, "sha1=", "", 1)
	computedSignatureHMAC := hmac.New(sha1.New, []byte(hubSecret))
	computedSignatureHMAC.Write(reqBody)
	computedSig := fmt.Sprintf("%x", computedSignatureHMAC.Sum(nil))
	if icSig != computedSig {
		jsonhttp.JSONForbiddenError(w, "Invalid request signature", "")
		return errors.New("invalid signature")
	}
	err := json.Unmarshal(reqBody, &outStruct)
	if err != nil {
		jsonhttp.JSONBadRequestError(w, "Invalid JSON", "")
		return err
	}
	if !icIsCheckableRequest(outStruct) {
		return nil
	}
	method := reflect.ValueOf(outStruct).MethodByName("Parameters").Interface().(func() error)
	err = method()
	if err != nil {
		jsonhttp.JSONBadRequestError(w, "", err.Error())
		return err
	}
	return nil

}

func icIsCheckableRequest(checkAgainst interface{}) bool {
	reader := reflect.TypeOf((*GHCheckableRequest)(nil)).Elem()
	return reflect.TypeOf(checkAgainst).Implements(reader)
}

// GHCheckableRequest defines an interface for request payloads that can be checked with the jsonhttp checker. See JSONDecodeAndCatchForAPI for the usage
type GHCheckableRequest interface {
	Parameters() error
}
