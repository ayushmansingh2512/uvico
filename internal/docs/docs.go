package docs

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	driveService *drive.Service
	docsService  *docs.Service
)

// InitServices initializes Google Drive and Docs clients using service-account.json
func InitServices(ctx context.Context) error {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		credsPath = "service-account.json"
	}

	if _, err := os.Stat(credsPath); os.IsNotExist(err) {
		return fmt.Errorf("credentials file not found at %s", credsPath)
	}

	opts := []option.ClientOption{
		option.WithCredentialsFile(credsPath),
		option.WithScopes(
			docs.DocumentsScope,
			docs.DriveScope,
			drive.DriveScope,
			drive.DriveFileScope,
		),
	}

	dService, err := docs.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Google Docs service: %w", err)
	}

	drService, err := drive.NewService(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Google Drive service: %w", err)
	}

	docsService = dService
	driveService = drService
	return nil
}

// getTargetFolderID searches for any folder shared with the service account or parses custom folder ID / URL
func getTargetFolderID(ctx context.Context, customFolderID string) string {
	customFolderID = strings.TrimSpace(customFolderID)
	if customFolderID != "" {
		if strings.Contains(customFolderID, "folders/") {
			parts := strings.Split(customFolderID, "folders/")
			if len(parts) > 1 {
				customFolderID = strings.Split(parts[1], "?")[0]
			}
		}
		if customFolderID != "" {
			return customFolderID
		}
	}

	folderID := os.Getenv("GOOGLE_DRIVE_FOLDER_ID")
	if folderID != "" {
		return folderID
	}

	if driveService == nil {
		return ""
	}

	// Search for any folders shared with the service account
	queries := []string{
		"mimeType = 'application/vnd.google-apps.folder' and trashed = false and sharedWithMe = true",
		"mimeType = 'application/vnd.google-apps.folder' and trashed = false",
	}

	for _, q := range queries {
		r, err := driveService.Files.List().
			Q(q).
			Fields("files(id, name)").
			SupportsAllDrives(true).
			IncludeItemsFromAllDrives(true).
			PageSize(10).
			Context(ctx).
			Do()
		if err == nil && len(r.Files) > 0 {
			log.Printf("[Google Docs] Auto-detected shared folder: '%s' (ID: %s)", r.Files[0].Name, r.Files[0].Id)
			return r.Files[0].Id
		}
	}

	return ""
}

// findTemplateDoc searches for any existing Google Doc inside the target folder to copy or update
func findTemplateDoc(ctx context.Context, folderID string) string {
	if driveService == nil || folderID == "" {
		return ""
	}

	q := fmt.Sprintf("'%s' in parents and mimeType = 'application/vnd.google-apps.document' and trashed = false", folderID)
	r, err := driveService.Files.List().
		Q(q).
		Fields("files(id, name)").
		SupportsAllDrives(true).
		IncludeItemsFromAllDrives(true).
		PageSize(5).
		Context(ctx).
		Do()
	if err == nil && len(r.Files) > 0 {
		log.Printf("[Google Docs] Found existing doc '%s' (ID: %s) in folder %s", r.Files[0].Name, r.Files[0].Id, folderID)
		return r.Files[0].Id
	}
	return ""
}

// CreateAndShareDoc creates or updates a Google Doc and returns the live URL
func CreateAndShareDoc(ctx context.Context, ownerEmail, folderID, title, content string) (string, error) {
	if driveService == nil || docsService == nil {
		if err := InitServices(ctx); err != nil {
			return "", err
		}
	}

	if strings.TrimSpace(title) == "" {
		title = "Generated Document"
	}

	targetFolderID := getTargetFolderID(ctx, folderID)
	var docID string

	// Method A: Check for existing document in the user's shared folder
	templateID := findTemplateDoc(ctx, targetFolderID)
	if templateID != "" {
		// Try to clone first
		copyMetadata := &drive.File{
			Name:    title,
			Parents: []string{targetFolderID},
		}
		copiedFile, copyErr := driveService.Files.Copy(templateID, copyMetadata).
			SupportsAllDrives(true).
			Fields("id, name").
			Context(ctx).
			Do()

		if copyErr == nil && copiedFile != nil && copiedFile.Id != "" {
			docID = copiedFile.Id
			log.Printf("[Google Docs] Successfully cloned doc ID: %s", docID)
		} else {
			// Update the existing document in folder directly (zero quota required)
			log.Printf("[Google Docs] Using and updating document directly in folder: %s", templateID)
			docID = templateID
			_, _ = driveService.Files.Update(docID, &drive.File{Name: title}).
				SupportsAllDrives(true).
				Context(ctx).
				Do()
		}
	}

	// Method B: Create directly via Docs API if no folder doc was found
	if docID == "" {
		log.Printf("[Google Docs] Creating doc via Docs API with title: '%s'", title)
		doc, err := docsService.Documents.Create(&docs.Document{
			Title: title,
		}).Context(ctx).Do()

		if err == nil && doc != nil && doc.DocumentId != "" {
			docID = doc.DocumentId
			log.Printf("[Google Docs] Successfully created doc ID via Docs API: %s", docID)

			if targetFolderID != "" {
				_, _ = driveService.Files.Update(docID, nil).
					AddParents(targetFolderID).
					SupportsAllDrives(true).
					Context(ctx).
					Do()
			}
		} else {
			// Method C: Create in Drive parent folder
			fileMetadata := &drive.File{
				Name:     title,
				MimeType: "application/vnd.google-apps.document",
			}
			if targetFolderID != "" {
				fileMetadata.Parents = []string{targetFolderID}
			}

			file, driveErr := driveService.Files.Create(fileMetadata).
				Fields("id, name, webViewLink").
				SupportsAllDrives(true).
				Context(ctx).
				Do()

			if driveErr != nil {
				log.Printf("[Google Docs] Drive.Create error: %v", driveErr)
				return "", fmt.Errorf("Please create 1 blank Google Doc inside your 'copilot' folder so the bot can use it (Error: %v)", driveErr)
			}
			docID = file.Id
		}
	}

	// 2. Clear old content and insert full generated content via Google Docs API
	if strings.TrimSpace(content) != "" && docID != "" {
		var requests []*docs.Request

		docObj, getErr := docsService.Documents.Get(docID).Context(ctx).Do()
		if getErr == nil && docObj != nil && docObj.Body != nil && len(docObj.Body.Content) > 0 {
			endIndex := docObj.Body.Content[len(docObj.Body.Content)-1].EndIndex
			if endIndex > 2 {
				requests = append(requests, &docs.Request{
					DeleteContentRange: &docs.DeleteContentRangeRequest{
						Range: &docs.Range{
							StartIndex: 1,
							EndIndex:   endIndex - 1,
						},
					},
				})
			}
		}

		requests = append(requests, &docs.Request{
			InsertText: &docs.InsertTextRequest{
				Location: &docs.Location{Index: 1},
				Text:     fmt.Sprintf("%s\n\n%s", title, content),
			},
		})

		_, err := docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
			Requests: requests,
		}).Context(ctx).Do()

		if err != nil {
			log.Printf("[Google Docs] Notice updating text in doc %s: %v", docID, err)
		} else {
			log.Printf("[Google Docs] Successfully updated text content in doc ID: %s", docID)
		}
	}

	// 3. Grant permissions via Drive API
	permAnyone := &drive.Permission{
		Type: "anyone",
		Role: "writer",
	}
	_, _ = driveService.Permissions.Create(docID, permAnyone).SupportsAllDrives(true).Context(ctx).Do()

	if ownerEmail != "" && strings.Contains(ownerEmail, "@") {
		permUser := &drive.Permission{
			Type:         "user",
			Role:         "writer",
			EmailAddress: ownerEmail,
		}
		_, _ = driveService.Permissions.Create(docID, permUser).SupportsAllDrives(true).SendNotificationEmail(false).Context(ctx).Do()
	}

	docURL := fmt.Sprintf("https://docs.google.com/document/d/%s/edit", docID)
	return docURL, nil
}
