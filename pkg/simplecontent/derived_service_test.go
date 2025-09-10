package simplecontent_test

import (
    "context"
    "testing"

    "github.com/google/uuid"
    simplecontent "github.com/tendant/simple-content/pkg/simplecontent"
    memoryrepo "github.com/tendant/simple-content/pkg/simplecontent/repo/memory"
)

func TestCreateDerived_InferDerivationTypeFromVariant(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    derived, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
        ParentID: parent.ID,
        OwnerID: parent.OwnerID,
        TenantID: parent.TenantID,
        Variant:  "thumbnail_256", // derivation_type omitted; should infer "thumbnail"
    })
    if err != nil { t.Fatalf("create derived: %v", err) }
    if got := derived.DerivationType; got != "thumbnail" {
        t.Fatalf("expected derivation_type 'thumbnail', got %q", got)
    }
}

func TestListDerivedAndGetRelationship(t *testing.T) {
    svc := mustService(t)
    ctx := context.Background()

    parent, err := svc.CreateContent(ctx, simplecontent.CreateContentRequest{
        OwnerID: uuid.New(), TenantID: uuid.New(), Name: "parent",
    })
    if err != nil { t.Fatalf("create parent: %v", err) }

    // create two variants
    for _, v := range []string{"thumbnail_128","thumbnail_256"} {
        if _, err := svc.CreateDerivedContent(ctx, simplecontent.CreateDerivedContentRequest{
            ParentID: parent.ID,
            OwnerID: parent.OwnerID,
            TenantID: parent.TenantID,
            Variant:  v,
        }); err != nil {
            t.Fatalf("create derived %s: %v", v, err)
        }
    }

    rels, err := svc.ListDerivedByParent(ctx, parent.ID)
    if err != nil { t.Fatalf("list derived: %v", err) }
    if len(rels) < 2 { t.Fatalf("expected >=2 derived, got %d", len(rels)) }

    // Check we can resolve one relationship by content id
    rel, err := svc.GetDerivedRelationshipByContentID(ctx, rels[0].ContentID)
    if err != nil { t.Fatalf("get relationship: %v", err) }
    if rel.ParentID != parent.ID { t.Fatalf("parent mismatch") }
}

func mustService(t *testing.T) simplecontent.Service {
    t.Helper()
    repo := memoryrepo.New()
    svc, err := simplecontent.New(simplecontent.WithRepository(repo))
    if err != nil { t.Fatalf("service new: %v", err) }
    return svc
}
