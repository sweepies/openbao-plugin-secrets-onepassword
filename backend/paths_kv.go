package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	onepassword "github.com/1password/onepassword-sdk-go"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

// errAmbiguousField is returned when two fields of one item would map
// to the same kv/ key even after section qualification.
var errAmbiguousField = errors.New("op: ambiguous field label")

// pathKV defines `kv/<vault>/<item...>`: an item's fields as a flat
// map of label to value, the shape KV clients such as `bao kv get
// -field=<label>` read. Addressing matches item/.
func pathKV(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `kv/(?P<vault>[^/]+)/(?P<item>.+)`,

		Fields: map[string]*framework.FieldSchema{
			"vault": {
				Type:        framework.TypeString,
				Description: "1Password vault name or ID.",
			},
			"item": {
				Type:        framework.TypeString,
				Description: "1Password item ID, exact title, or path_split address.",
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathKVRead,
				Summary:  "Read a 1Password item's fields as a label-to-value map.",
			},
		},

		HelpSynopsis: "Read a 1Password item as a flat key/value map.",
		HelpDescription: "Returns each field's value keyed by its label. A label that occurs more than once in the item " +
			"is keyed \"<section>.<label>\" instead; a collision that remains is an error (read field/ instead).",
	}
}

func (b *Backend) pathKVRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	vaultAddr := data.Get("vault").(string)
	itemAddr := data.Get("item").(string)

	vaultID, err := b.resolveVaultAddress(vaultAddr)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	segs := strings.Split(itemAddr, "/")
	r := b.getOrCreateReplica(vaultID)
	resolve := func() (string, error) { return resolveItemFromSegments(r, segs) }

	itemID, found, err := b.ensureItemFresh(ctx, vaultID, itemAddr, resolve)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}
	if !found {
		return logical.ErrorResponse(errNotFound.Error()), nil
	}

	if cfg := b.currentConfig(); cfg != nil {
		if serr := checkServeStaleAllowed(cfg, r, time.Now()); serr != nil {
			return logical.ErrorResponse(serr.Error()), nil
		}
	}

	item, ok := r.body(itemID)
	if !ok {
		return logical.ErrorResponse(errNotFound.Error()), nil
	}

	respData, err := kvResponseData(item)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}
	return &logical.Response{Data: respData}, nil
}

// kvResponseData maps an item's fields to label → value. Labels that
// occur once are used as is; repeated labels are qualified with their
// section title so a section's "password" doesn't shadow the item's
// own. Unsectioned repeats, or repeats within one section, fail.
func kvResponseData(item onepassword.Item) (map[string]interface{}, error) {
	sectionTitles := make(map[string]string, len(item.Sections))
	for _, s := range item.Sections {
		sectionTitles[s.ID] = s.Title
	}
	labelCounts := make(map[string]int, len(item.Fields))
	for _, f := range item.Fields {
		labelCounts[f.Title]++
	}

	out := make(map[string]interface{}, len(item.Fields))
	for _, f := range item.Fields {
		key := f.Title
		if labelCounts[f.Title] > 1 {
			if f.SectionID == nil || sectionTitles[*f.SectionID] == "" {
				return nil, fmt.Errorf("%w %q in item %q", errAmbiguousField, f.Title, item.Title)
			}
			key = sectionTitles[*f.SectionID] + "." + f.Title
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("%w %q in item %q", errAmbiguousField, key, item.Title)
		}
		out[key] = f.Value
	}
	return out, nil
}
