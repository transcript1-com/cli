package api

import "context"

type uploadDescription struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
}

type createUploadBody struct {
	Upload   uploadDescription `json:"upload"`
	Language string            `json:"language"`
	Diarize  bool              `json:"diarize"`
}

func (c *Client) CreateUpload(ctx context.Context, idemKey, filename string, size int64, mime string, opt CreateOptions) (*Transcription, error) {
	body := createUploadBody{
		Upload: uploadDescription{
			Filename: filename,
			Size:     size,
			MimeType: mime,
		},
		Language: opt.getLanguage(),
		Diarize:  opt.Diarize,
	}
	res, err := c.postJSON(ctx, "/v1/transcriptions", nil, idemKey, body)
	if err != nil {
		return nil, err
	}

	return decodeTranscription(res)
}
