package models

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Note struct {
	HasID
	Title        string `gorm:"index;not null"`
	Description  string `gorm:"index"`
	Body         string `gorm:"longtext"`
	LabelId      int    `gorm:"index;not null"`
	ParentNoteId int    `gorm:"index"`
	Label        Label
	RemindDate   *time.Time `gorm:"index;default:null"`
	HasUserId
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (input *Note) validate(db *gorm.DB, userId int, id int) error {

	userFilter := NewFilter("user_id = ?", userId)
	return Validate(db, NewExistsRule("labels", input.LabelId, "label not found", userFilter),
		NewUniqueRule("notes", "title", input.Title, id, "duplicate title", userFilter),
	)
}

type NoteResource struct {
	Id           int
	Title        string
	Description  string
	Body         string
	LabelId      int
	LabelName    string
	ParentNoteId int
	RemindDate   MyDate
	CreatedAt    MyDateTime
	UpdatedAt    MyDateTime
}

type NoteService struct {
	db *gorm.DB
}

func (s *NoteService) fetch(ctx context.Context, userId int, id int, preloads ...string) (*Note, error) {
	return first[Note](s.db.WithContext(ctx), userId, id, preloads...)
}

func (s *NoteService) Create(ctx context.Context, userId int, input *Note) (*Note, error) {

	if err := input.validate(s.db.WithContext(ctx), userId, 0); err != nil {
		return nil, err
	}
	input.UserId = userId

	err := s.db.WithContext(ctx).Create(&input).Error
	return input, err
}

func (s *NoteService) Update(ctx context.Context, userId int, id int, input *Note) (*Note, error) {

	if err := input.validate(s.db.WithContext(ctx), userId, id); err != nil {
		return nil, err
	}
	note, err := s.fetch(ctx, userId, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"Title":       input.Title,
		"Description": input.Description,
		"Body":        input.Body,
		"LabelId":     input.LabelId,
		"RemindDate":  input.RemindDate,
	}
	if err := s.db.WithContext(ctx).Model(&note).Updates(updates).Error; err != nil {
		return nil, err
	}
	return note, nil
}

func (s *NoteService) UpdateBody(ctx context.Context, userId int, id int, body string) (*NoteResource, error) {
	note, err := s.fetch(ctx, userId, id, "Label")
	if err != nil {
		return nil, err
	}
	err = s.db.WithContext(ctx).Model(&note).UpdateColumn("body", body).Error
	if err != nil {
		return nil, err
	}
	return s.ConvertToResource(note), nil
}

func (s *NoteService) Delete(ctx context.Context, userId int, id int) (*Note, error) {
	note, err := s.fetch(ctx, userId, id)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Delete(&note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

func (s *NoteService) ConvertToResource(note *Note) *NoteResource {
	var remindDate MyDate
	if note.RemindDate != nil {
		remindDate = MyDate{*note.RemindDate}
	}
	res := NoteResource{
		Id:          note.Id,
		Title:       note.Title,
		Description: note.Description,
		Body:        note.Body,
		LabelId:     note.LabelId,
		LabelName:   note.Label.Name,
		RemindDate:  remindDate,
		CreatedAt:   MyDateTime{note.CreatedAt},
		UpdatedAt:   MyDateTime{note.UpdatedAt},
	}
	return &res
}

func (s *NoteService) Get(ctx context.Context, userId int, id int) (*NoteResource, error) {

	note, err := first[Note](s.db.WithContext(ctx), userId, id, "Label")
	if err != nil {
		return nil, err
	}
	res := s.ConvertToResource(note)
	return res, nil
}

func (s *NoteService) ListNotes(ctx context.Context, userId int) ([]*NoteResource, error) {
	notes, err := find[Note](s.db.WithContext(ctx), userId, "Label")
	if err != nil {
		return nil, err
	}
	resCollection := make([]*NoteResource, 0, len(notes))
	for _, n := range notes {
		resCollection = append(resCollection, s.ConvertToResource(&n))
	}

	return resCollection, nil
}
