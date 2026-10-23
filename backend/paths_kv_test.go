package backend

import (
	"context"
	"errors"
	"reflect"
	"testing"

	onepassword "github.com/1password/onepassword-sdk-go"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func readKV(t *testing.T, b *Backend, storage logical.Storage, vault, item string) *logical.Response {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "kv/" + vault + "/" + item,
		Storage:   storage,
	})
	if err != nil {
		t.Fatalf("read kv %s/%s: unexpected error: %v", vault, item, err)
	}
	return resp
}

func TestKVReadReturnsFieldMap(t *testing.T) {
	b, storage, _ := setupItemBackend(t, nil)

	resp := readKV(t, b, storage, "Infra", "postgres")
	if resp == nil || resp.IsError() {
		t.Fatalf("resp=%+v, want success", resp)
	}
	want := map[string]interface{}{"password": "hunter2"}
	if !reflect.DeepEqual(resp.Data, want) {
		t.Errorf("data = %v, want %v", resp.Data, want)
	}
}

func TestKVReadByID(t *testing.T) {
	b, storage, _ := setupItemBackend(t, nil)

	resp := readKV(t, b, storage, "v1", "i2")
	if resp == nil || resp.IsError() {
		t.Fatalf("resp=%+v, want success", resp)
	}
	if resp.Data["password"] != "swordfish" {
		t.Errorf("password = %v, want swordfish", resp.Data["password"])
	}
}

func TestKVReadUnknownItem(t *testing.T) {
	b, storage, _ := setupItemBackend(t, map[string]interface{}{"negative_cache_ttl": "0s"})

	resp := readKV(t, b, storage, "Infra", "missing")
	if resp == nil || !resp.IsError() {
		t.Fatalf("resp=%+v, want an error response", resp)
	}
}

func sectionID(id string) *string { return &id }

func TestKVResponseDataRejectsUnsectionedRepeat(t *testing.T) {
	item := onepassword.Item{
		Title:    "github",
		Sections: []onepassword.ItemSection{{ID: "s1", Title: "backup"}},
		Fields: []onepassword.ItemField{
			{Title: "password", Value: "main"},
			{Title: "password", Value: "spare", SectionID: sectionID("s1")},
		},
	}

	if _, err := kvResponseData(item); !errors.Is(err, errAmbiguousField) {
		t.Fatalf("err = %v, want errAmbiguousField", err)
	}
}

func TestKVResponseDataSectionQualified(t *testing.T) {
	item := onepassword.Item{
		Title: "github",
		Sections: []onepassword.ItemSection{
			{ID: "s1", Title: "primary"},
			{ID: "s2", Title: "backup"},
		},
		Fields: []onepassword.ItemField{
			{Title: "username", Value: "sweepies"},
			{Title: "password", Value: "main", SectionID: sectionID("s1")},
			{Title: "password", Value: "spare", SectionID: sectionID("s2")},
		},
	}

	got, err := kvResponseData(item)
	if err != nil {
		t.Fatalf("kvResponseData: %v", err)
	}
	want := map[string]interface{}{
		"username":         "sweepies",
		"primary.password": "main",
		"backup.password":  "spare",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("data = %v, want %v", got, want)
	}
}

func TestKVResponseDataRejectsRepeatWithinSection(t *testing.T) {
	item := onepassword.Item{
		Title:    "github",
		Sections: []onepassword.ItemSection{{ID: "s1", Title: "backup"}},
		Fields: []onepassword.ItemField{
			{Title: "password", Value: "a", SectionID: sectionID("s1")},
			{Title: "password", Value: "b", SectionID: sectionID("s1")},
		},
	}

	if _, err := kvResponseData(item); !errors.Is(err, errAmbiguousField) {
		t.Fatalf("err = %v, want errAmbiguousField", err)
	}
}
