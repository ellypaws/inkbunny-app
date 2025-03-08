package app

import "testing"

func TestLabel_MarshalJSON(t *testing.T) {
	type fields struct {
		Label string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "Test MarshalJSON",
			fields: fields{
				Label: "ai_generated",
			},
			want:    []byte(`"ai generated"`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Label(tt.fields.Label)
			got, err := l.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Label.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != string(tt.want) {
				t.Errorf("Label.MarshalJSON() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestUnmarshalMails(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    Mails
		wantErr bool
	}{
		{
			name: "Test MarshalMails",
			args: args{
				data: []byte(`[{"id":"123","name":"user","email":"email","subject":"subject","text":"text","html":"html","date":"date","read":false,"labels":["ai_generated","ai_assisted"]}]`),
			},
			want: Mails{
				{
					SubmissionID: "123",
					Username:     "user",
					Link:         "email",
					Title:        "subject",
					Description:  "text",
					Html:         "html",
					Date:         "date",
					Read:         false,
					Labels:       []Label{"ai_generated", "ai_assisted"},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalMails(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalMails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got[0].SubmissionID != tt.want[0].SubmissionID {
				t.Errorf("UnmarshalMails() = %v, want %v", got[0].SubmissionID, tt.want[0].SubmissionID)
			}
		})
	}
}
