package negroni

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func TestNegroniRun(t *testing.T) {
	// just test that Run doesn't bomb
	go New().Run(":3000")
}

func TestNegroniWith(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n1 := New()
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result = "one"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "two"
		next(rw, r)
	}))

	n1.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 2, len(n1.Handlers()))
	expect(t, result, "onetwo")

	n2 := n1.With(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "three"
		next(rw, r)
	}))

	// Verify that n1 was left intact and not modified.
	n1.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 2, len(n1.Handlers()))
	expect(t, result, "onetwo")

	n2.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 3, len(n2.Handlers()))
	expect(t, result, "onetwothree")
}

func TestNegroniWithIssueCase(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n1 := New()
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result = "one"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "two"
		next(rw, r)
	}))

	// Using large number of middleware will cause append() to use the old slice as destination
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "three"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "four"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "five"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "six"
		next(rw, r)
	}))
	n1.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "seven"
		next(rw, r)
	}))

	n1.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 7, len(n1.Handlers()))
	expect(t, result, "onetwothreefourfivesixseven")

	n2 := n1.With(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "eight"
		next(rw, r)
	}))

	n3 := n1.With(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "nine"
		next(rw, r)
	}))

	n4 := n1.With(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "ten"
		next(rw, r)
	}))


	// Finally we also add one remaining UseHandlerFunc call for the router(/subrouters)
	n2.UseHandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Represents router/subrouter
	})

	n3.UseHandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Represents router/subrouter
	})

	n4.UseHandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// Represents router/subrouter
	})


	// Verify that n1 was left intact and not modified.
	n1.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 7, len(n1.Handlers()))
	expect(t, result, "onetwothreefourfivesixseven")

	n2.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 9, len(n2.Handlers()))
	expect(t, result, "onetwothreefourfivesixseveneight")

	n3.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 9, len(n3.Handlers()))
	expect(t, result, "onetwothreefourfivesixsevennine")

	n4.ServeHTTP(response, (*http.Request)(nil))
	expect(t, 9, len(n4.Handlers()))
	expect(t, result, "onetwothreefourfivesixseventen")
}

func TestNegroniServeHTTP(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n := New()
	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "foo"
		next(rw, r)
		result += "ban"
	}))
	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "bar"
		next(rw, r)
		result += "baz"
	}))
	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		result += "bat"
		rw.WriteHeader(http.StatusBadRequest)
	}))

	n.ServeHTTP(response, (*http.Request)(nil))

	expect(t, result, "foobarbatbazban")
	expect(t, response.Code, http.StatusBadRequest)
}

// Ensures that a Negroni middleware chain
// can correctly return all of its handlers.
func TestHandlers(t *testing.T) {
	response := httptest.NewRecorder()
	n := New()
	handlers := n.Handlers()
	expect(t, 0, len(handlers))

	n.Use(HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		rw.WriteHeader(http.StatusOK)
	}))

	// Expects the length of handlers to be exactly 1
	// after adding exactly one handler to the middleware chain
	handlers = n.Handlers()
	expect(t, 1, len(handlers))

	// Ensures that the first handler that is in sequence behaves
	// exactly the same as the one that was registered earlier
	handlers[0].ServeHTTP(response, (*http.Request)(nil), nil)
	expect(t, response.Code, http.StatusOK)
}

func TestNegroni_Use_Nil(t *testing.T) {
	defer func() {
		err := recover()
		if err == nil {
			t.Errorf("Expected negroni.Use(nil) to panic, but it did not")
		}
	}()

	n := New()
	n.Use(nil)
}

func TestDetectAddress(t *testing.T) {
	if detectAddress() != DefaultAddress {
		t.Error("Expected the DefaultAddress")
	}

	if detectAddress(":6060") != ":6060" {
		t.Error("Expected the provided address")
	}

	os.Setenv("PORT", "8080")
	if detectAddress() != ":8080" {
		t.Error("Expected the PORT env var with a prefixed colon")
	}
}

func voidHTTPHandlerFunc(rw http.ResponseWriter, r *http.Request) {
	// Do nothing
}

// Test for function Wrap
func TestWrap(t *testing.T) {
	response := httptest.NewRecorder()

	handler := Wrap(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(response, (*http.Request)(nil), voidHTTPHandlerFunc)

	expect(t, response.Code, http.StatusOK)
}

// Test for function WrapFunc
func TestWrapFunc(t *testing.T) {
	response := httptest.NewRecorder()

	// WrapFunc(f) equals Wrap(http.HandlerFunc(f)), it's simpler and usefull.
	handler := WrapFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(response, (*http.Request)(nil), voidHTTPHandlerFunc)

	expect(t, response.Code, http.StatusOK)
}

type MockMiddleware struct {
	Counter int
	Name	string
}

func (m *MockMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	m.Counter++
	next(rw, r)
}

func newMiddlewareStruct(name string) *MockMiddleware {
	return &MockMiddleware{Name: name}
}

func sameHandlers(handlers []Handler, handlers2 []Handler) bool {
	for i, v := range handlers {
		sf1 := reflect.ValueOf(v)
		sf2 := reflect.ValueOf(handlers2[i])

		if sf1 != sf2 {
			return false
		}
	}
	return true
}


func TestHandlerVerification(t *testing.T) {
	n := New()

	mid1 := newMiddlewareStruct("mid1")
	mid2 := newMiddlewareStruct("mid2")
	mid3 := newMiddlewareStruct("mid3")
	mid4 := newMiddlewareStruct("mid4")
	mid5 := newMiddlewareStruct("mid5")
	mid6 := newMiddlewareStruct("mid6")
	mid7 := newMiddlewareStruct("mid7")

	n.Use(mid1)
	n.Use(mid2)
	n.Use(mid3)
	n.Use(mid4)
	n.Use(mid5)
	n.Use(mid6)
	n.Use(mid7)

	mid8 := newMiddlewareStruct("mid8")
	subNeg1 := n.With(mid8)

	mid9 := newMiddlewareStruct("mid9")
	subNeg2 := n.With(mid9)

	mid10 := newMiddlewareStruct("mid10")
	subNeg3 := n.With(mid10)

	fmt.Println(subNeg1)
	fmt.Println(subNeg3)

	if !sameHandlers(subNeg1.handlers, []Handler{mid1, mid2, mid3, mid4, mid5, mid6, mid7, mid8}) {
		t.Error("Handlers not the same")
	}
	if !sameHandlers(subNeg2.handlers, []Handler{mid1, mid2, mid3, mid4, mid5, mid6, mid7, mid9}) {
		t.Error("Handlers not the same")
	}
	if !sameHandlers(subNeg3.handlers, []Handler{mid1, mid2, mid3, mid4, mid5, mid6, mid7, mid10}) {
		t.Error("Handlers not the same")
	}

	subNeg2.UseHandlerFunc(func (rw http.ResponseWriter, r *http.Request){
		// final handler // router
	})

	response := httptest.NewRecorder()
	subNeg2.ServeHTTP(response, (*http.Request)(nil))

	testCounter(t, mid1, 1, "mid1")
	testCounter(t, mid2, 1, "mid2")
	testCounter(t, mid3, 1, "mid3")
	testCounter(t, mid4, 1, "mid4")
	testCounter(t, mid5, 1, "mid5")
	testCounter(t, mid6, 1, "mid6")
	testCounter(t, mid7, 1, "mid7")

	// mid8 is part of midSubOne and should not be called
	testCounter(t, mid8, 0, "mid8")
	// mid9 is part of midSubTwo and should not be called
	testCounter(t, mid9, 1, "mid9")
	// mid10 is part of unused midSubThree and should not be called
	testCounter(t, mid10, 0, "mid10")



}

func testCounter(t *testing.T, counter *MockMiddleware, expected int, identifier string) {
	if counter.Counter != expected {
		t.Errorf("Expected %d call to middleware instead %d (%s)", expected, counter.Counter, identifier)
	}
}