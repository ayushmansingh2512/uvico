package docs

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
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

// CreateAndShareDoc creates or updates a Google Doc and returns the live URL (default template)
func CreateAndShareDoc(ctx context.Context, ownerEmail, folderID, title, content string) (string, error) {
	return CreateAndShareDocWithOptions(ctx, ownerEmail, folderID, title, content, "")
}

// CreateAndShareDocWithOptions creates or updates a Google Doc with template styling (e.g. "kiet")
func CreateAndShareDocWithOptions(ctx context.Context, ownerEmail, folderID, title, content, templateType string) (string, error) {
	if driveService == nil || docsService == nil {
		if err := InitServices(ctx); err != nil {
			return "", err
		}
	}

	// Strip all emojis from title and content for a clean, formal document
	emojiPattern := regexp.MustCompile(`[\x{1F000}-\x{1FFFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{2300}-\x{23FF}\x{2B50}-\x{2B55}]`)
	title = strings.TrimSpace(emojiPattern.ReplaceAllString(title, ""))
	content = strings.TrimSpace(emojiPattern.ReplaceAllString(content, ""))

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
		// First: clear existing doc content
		docObj, getErr := docsService.Documents.Get(docID).Context(ctx).Do()
		if getErr == nil && docObj != nil && docObj.Body != nil && len(docObj.Body.Content) > 0 {
			endIndex := docObj.Body.Content[len(docObj.Body.Content)-1].EndIndex
			if endIndex > 2 {
				_, _ = docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
					Requests: []*docs.Request{
						{
							DeleteContentRange: &docs.DeleteContentRangeRequest{
								Range: &docs.Range{
									StartIndex: 1,
									EndIndex:   endIndex - 1,
								},
							},
						},
					},
				}).Context(ctx).Do()
			}
		}

		if strings.ToLower(templateType) == "kiet" {
			applyKIETDocumentFormat(ctx, docID, title, content)
		} else {
			// Standard document insertion with Narrow Margins (36 pt = 0.5 in)
			insertReqs := []*docs.Request{
				{
					UpdateDocumentStyle: &docs.UpdateDocumentStyleRequest{
						DocumentStyle: &docs.DocumentStyle{
							MarginTop:    &docs.Dimension{Magnitude: 36, Unit: "PT"},
							MarginBottom: &docs.Dimension{Magnitude: 36, Unit: "PT"},
							MarginLeft:   &docs.Dimension{Magnitude: 36, Unit: "PT"},
							MarginRight:  &docs.Dimension{Magnitude: 36, Unit: "PT"},
						},
						Fields: "marginTop,marginBottom,marginLeft,marginRight",
					},
				},
				{
					InsertText: &docs.InsertTextRequest{
						Location: &docs.Location{Index: 1},
						Text:     fmt.Sprintf("%s\n\n%s", title, content),
					},
				},
			}
			_, err := docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
				Requests: insertReqs,
			}).Context(ctx).Do()
			if err != nil {
				log.Printf("[Google Docs] Notice updating text in doc %s: %v", docID, err)
			}
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

// applyKIETDocumentFormat styles the document with official KIET letterhead, Times New Roman, and center alignment
func applyKIETDocumentFormat(ctx context.Context, docID, title, content string) {
	// Strip all emojis from title and content
	emojiPattern := regexp.MustCompile(`[\x{1F000}-\x{1FFFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{2300}-\x{23FF}\x{2B50}-\x{2B55}]`)
	title = strings.TrimSpace(emojiPattern.ReplaceAllString(title, ""))
	content = strings.TrimSpace(emojiPattern.ReplaceAllString(content, ""))

	// 1. Try to insert official KIET University Logo image at Index 1
	var imgInserted bool
	imgResp, imgErr := docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertInlineImage: &docs.InsertInlineImageRequest{
					Location: &docs.Location{Index: 1},
					Uri:      "https://manthan.kiet.edu/images/kietLogo.jpg",
					ObjectSize: &docs.Size{
						Height: &docs.Dimension{Magnitude: 65, Unit: "PT"},
						Width:  &docs.Dimension{Magnitude: 175, Unit: "PT"},
					},
				},
			},
		},
	}).Context(ctx).Do()

	if imgErr == nil && imgResp != nil {
		imgInserted = true
		log.Printf("[Google Docs] Successfully inserted official KIET logo image into doc %s", docID)
	} else {
		log.Printf("[Google Docs] Notice on logo image insert: %v (using center-aligned text header)", imgErr)
	}

	titleSection := title + "\n\n"
	bodySection := content + "\n"

	var textToInsert string
	var headerEnd int64 = 1

	if imgInserted {
		// When official logo image is inserted at top, insert title and body after image
		textToInsert = "\n\n" + titleSection + bodySection
		headerEnd = 3 // after image and newlines
	} else {
		// Fallback: Center-aligned official text header
		headerL1 := "KIET\n"
		headerL2 := "DEEMED TO BE UNIVERSITY\n"
		headerL3 := "DELHI-NCR, GHAZIABAD, UP (INDIA)\n"
		headerL4 := "(Under Section 3 of the UGC Act, 1956)\n\n"
		headerText := headerL1 + headerL2 + headerL3 + headerL4
		textToInsert = headerText + titleSection + bodySection
		headerEnd = 1 + int64(len([]rune(headerText)))
	}

	insertLoc := int64(1)
	if imgInserted {
		insertLoc = 2
	}

	// Insert document text
	_, err := docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
		Requests: []*docs.Request{
			{
				InsertText: &docs.InsertTextRequest{
					Location: &docs.Location{Index: insertLoc},
					Text:     textToInsert,
				},
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("[Google Docs] KIET insert text error: %v", err)
		return
	}

	// Build formatting requests
	var styleRequests []*docs.Request
	totalEnd := insertLoc + int64(len([]rune(textToInsert)))

	// 0. Set Narrow Margins (0.5 inch = 36 pt)
	styleRequests = append(styleRequests, &docs.Request{
		UpdateDocumentStyle: &docs.UpdateDocumentStyleRequest{
			DocumentStyle: &docs.DocumentStyle{
				MarginTop:    &docs.Dimension{Magnitude: 36, Unit: "PT"},
				MarginBottom: &docs.Dimension{Magnitude: 36, Unit: "PT"},
				MarginLeft:   &docs.Dimension{Magnitude: 36, Unit: "PT"},
				MarginRight:  &docs.Dimension{Magnitude: 36, Unit: "PT"},
			},
			Fields: "marginTop,marginBottom,marginLeft,marginRight",
		},
	})

	// A. Apply Times New Roman across the entire document
	styleRequests = append(styleRequests, &docs.Request{
		UpdateTextStyle: &docs.UpdateTextStyleRequest{
			Range: &docs.Range{
				StartIndex: 1,
				EndIndex:   totalEnd,
			},
			TextStyle: &docs.TextStyle{
				WeightedFontFamily: &docs.WeightedFontFamily{
					FontFamily: "Times New Roman",
				},
				FontSize: &docs.Dimension{
					Magnitude: 11,
					Unit:      "PT",
				},
				ForegroundColor: &docs.OptionalColor{
					Color: &docs.Color{
						RgbColor: &docs.RgbColor{Red: 0.12, Green: 0.12, Blue: 0.12},
					},
				},
			},
			Fields: "weightedFontFamily,fontSize,foregroundColor",
		},
	})

	// B. Center align the top header / logo
	if imgInserted {
		// Center align image paragraph
		styleRequests = append(styleRequests, &docs.Request{
			UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
				Range: &docs.Range{
					StartIndex: 1,
					EndIndex:   2,
				},
				ParagraphStyle: &docs.ParagraphStyle{
					Alignment: "CENTER",
					SpaceBelow: &docs.Dimension{
						Magnitude: 14,
						Unit:      "PT",
					},
				},
				Fields: "alignment,spaceBelow",
			},
		})
	} else {
		// Center align text header
		styleRequests = append(styleRequests, &docs.Request{
			UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
				Range: &docs.Range{
					StartIndex: 1,
					EndIndex:   headerEnd - 1,
				},
				ParagraphStyle: &docs.ParagraphStyle{
					Alignment: "CENTER",
					SpaceBelow: &docs.Dimension{
						Magnitude: 2,
						Unit:      "PT",
					},
				},
				Fields: "alignment,spaceBelow",
			},
		})

		// Style header lines center aligned
		h1Len := int64(len([]rune("KIET\n")))
		h2Len := int64(len([]rune("DEEMED TO BE UNIVERSITY\n")))
		h3Len := int64(len([]rune("DELHI-NCR, GHAZIABAD, UP (INDIA)\n")))
		h4Len := int64(len([]rune("(Under Section 3 of the UGC Act, 1956)\n\n")))

		// "KIET" -> Bold, 18pt, Navy Blue
		styleRequests = append(styleRequests, &docs.Request{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				Range: &docs.Range{
					StartIndex: 1,
					EndIndex:   1 + h1Len - 1,
				},
				TextStyle: &docs.TextStyle{
					Bold: true,
					FontSize: &docs.Dimension{
						Magnitude: 18,
						Unit:      "PT",
					},
					ForegroundColor: &docs.OptionalColor{
						Color: &docs.Color{
							RgbColor: &docs.RgbColor{Red: 0.043, Green: 0.145, Blue: 0.270},
						},
					},
				},
				Fields: "bold,fontSize,foregroundColor",
			},
		})

		// "DEEMED TO BE UNIVERSITY" -> Bold, 14pt, Brick Orange
		uStart := 1 + h1Len
		styleRequests = append(styleRequests, &docs.Request{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				Range: &docs.Range{
					StartIndex: uStart,
					EndIndex:   uStart + h2Len - 1,
				},
				TextStyle: &docs.TextStyle{
					Bold: true,
					FontSize: &docs.Dimension{
						Magnitude: 14,
						Unit:      "PT",
					},
					ForegroundColor: &docs.OptionalColor{
						Color: &docs.Color{
							RgbColor: &docs.RgbColor{Red: 0.851, Green: 0.325, Blue: 0.118},
						},
					},
				},
				Fields: "bold,fontSize,foregroundColor",
			},
		})

		// "DELHI-NCR..." -> Bold, 9pt, Slate
		dStart := uStart + h2Len
		styleRequests = append(styleRequests, &docs.Request{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				Range: &docs.Range{
					StartIndex: dStart,
					EndIndex:   dStart + h3Len - 1,
				},
				TextStyle: &docs.TextStyle{
					Bold: true,
					FontSize: &docs.Dimension{
						Magnitude: 9,
						Unit:      "PT",
					},
					ForegroundColor: &docs.OptionalColor{
						Color: &docs.Color{
							RgbColor: &docs.RgbColor{Red: 0.18, Green: 0.22, Blue: 0.28},
						},
					},
				},
				Fields: "bold,fontSize,foregroundColor",
			},
		})

		// "(Under Section 3...)" -> Charcoal, 8pt
		sStart := dStart + h3Len
		styleRequests = append(styleRequests, &docs.Request{
			UpdateTextStyle: &docs.UpdateTextStyleRequest{
				Range: &docs.Range{
					StartIndex: sStart,
					EndIndex:   sStart + h4Len - 2,
				},
				TextStyle: &docs.TextStyle{
					Bold: false,
					FontSize: &docs.Dimension{
						Magnitude: 8,
						Unit:      "PT",
					},
					ForegroundColor: &docs.OptionalColor{
						Color: &docs.Color{
							RgbColor: &docs.RgbColor{Red: 0.32, Green: 0.36, Blue: 0.42},
						},
					},
				},
				Fields: "bold,fontSize,foregroundColor",
			},
		})
	}

	// C. Style Title: Centered, 14.5pt, Bold
	titleStart := headerEnd
	if imgInserted {
		titleStart = 3
	}
	titleEnd := titleStart + int64(len([]rune(title)))
	styleRequests = append(styleRequests, &docs.Request{
		UpdateParagraphStyle: &docs.UpdateParagraphStyleRequest{
			Range: &docs.Range{
				StartIndex: titleStart,
				EndIndex:   titleEnd + 1,
			},
			ParagraphStyle: &docs.ParagraphStyle{
				Alignment: "CENTER",
				SpaceAbove: &docs.Dimension{
					Magnitude: 10,
					Unit:      "PT",
				},
				SpaceBelow: &docs.Dimension{
					Magnitude: 16,
					Unit:      "PT",
				},
			},
			Fields: "alignment,spaceAbove,spaceBelow",
		},
	})
	styleRequests = append(styleRequests, &docs.Request{
		UpdateTextStyle: &docs.UpdateTextStyleRequest{
			Range: &docs.Range{
				StartIndex: titleStart,
				EndIndex:   titleEnd,
			},
			TextStyle: &docs.TextStyle{
				Bold: true,
				FontSize: &docs.Dimension{
					Magnitude: 14.5,
					Unit:      "PT",
				},
				ForegroundColor: &docs.OptionalColor{
					Color: &docs.Color{
						RgbColor: &docs.RgbColor{Red: 0.08, Green: 0.08, Blue: 0.12},
					},
				},
			},
			Fields: "bold,fontSize,foregroundColor",
		},
	})

	// D. Format Body lines (headings, yellow highlight for E-Certificates, bold labels)
	bodyStart := titleStart + int64(len([]rune(titleSection)))
	bodyLines := strings.Split(content, "\n")
	currOffset := bodyStart

	for _, line := range bodyLines {
		lineRuneLen := int64(len([]rune(line)))
		trimmed := strings.TrimSpace(line)

		if trimmed != "" {
			lineStart := currOffset
			lineEnd := currOffset + lineRuneLen

			// Check for yellow highlight on E-Certificates
			if strings.Contains(strings.ToLower(trimmed), "e-certificate") || strings.Contains(strings.ToLower(trimmed), "e-certificates") {
				styleRequests = append(styleRequests, &docs.Request{
					UpdateTextStyle: &docs.UpdateTextStyleRequest{
						Range: &docs.Range{
							StartIndex: lineStart,
							EndIndex:   lineEnd,
						},
						TextStyle: &docs.TextStyle{
							Bold: true,
							BackgroundColor: &docs.OptionalColor{
								Color: &docs.Color{
									RgbColor: &docs.RgbColor{Red: 1.0, Green: 0.95, Blue: 0.0},
								},
							},
						},
						Fields: "bold,backgroundColor",
					},
				})
			} else if strings.HasPrefix(trimmed, "Respected") ||
				strings.HasPrefix(trimmed, "Key Details:") ||
				strings.HasPrefix(trimmed, "Exciting Cash Prizes") ||
				strings.HasPrefix(trimmed, "Guaranteed Rewards") ||
				strings.HasPrefix(trimmed, "Cash Prizes") ||
				strings.HasPrefix(trimmed, "What Comes Next?") ||
				strings.HasPrefix(trimmed, "Warm Regards,") ||
				strings.HasPrefix(trimmed, "Dr. Preeti Chitkara") ||
				strings.HasPrefix(trimmed, "Dean") ||
				strings.HasPrefix(trimmed, "Department of PR") ||
				strings.HasPrefix(trimmed, "KIET Deemed to be University") ||
				strings.HasPrefix(trimmed, "For Registration and Further Information") ||
				strings.HasSuffix(trimmed, "Round") ||
				strings.HasSuffix(trimmed, "Finale") ||
				strings.HasPrefix(trimmed, "Stage ") {
				// Make section headings and signature lines bold
				styleRequests = append(styleRequests, &docs.Request{
					UpdateTextStyle: &docs.UpdateTextStyleRequest{
						Range: &docs.Range{
							StartIndex: lineStart,
							EndIndex:   lineEnd,
						},
						TextStyle: &docs.TextStyle{
							Bold: true,
						},
						Fields: "bold",
					},
				})
			} else if strings.HasPrefix(trimmed, "•") || strings.HasPrefix(trimmed, "-") {
				// Highlight bold key before colon if present (e.g. "• Eligibility: Classes XI & XII")
				if colonIdx := strings.Index(line, ":"); colonIdx > 0 {
					labelRuneLen := int64(len([]rune(line[:colonIdx+1])))
					styleRequests = append(styleRequests, &docs.Request{
						UpdateTextStyle: &docs.UpdateTextStyleRequest{
							Range: &docs.Range{
								StartIndex: lineStart,
								EndIndex:   lineStart + labelRuneLen,
							},
							TextStyle: &docs.TextStyle{
								Bold: true,
							},
							Fields: "bold",
						},
					})
				}
			}
		}

		// +1 for newline character
		currOffset += lineRuneLen + 1
	}

	// Execute style update batch
	if len(styleRequests) > 0 {
		_, err = docsService.Documents.BatchUpdate(docID, &docs.BatchUpdateDocumentRequest{
			Requests: styleRequests,
		}).Context(ctx).Do()
		if err != nil {
			log.Printf("[Google Docs] KIET styling batch error: %v", err)
		} else {
			log.Printf("[Google Docs] Successfully applied KIET University styling & Times New Roman to doc %s", docID)
		}
	}
}

