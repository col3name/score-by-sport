package postgres

import (
	errBase "errors"
	"github.com/col3name/lines/pkg/common/application/errors"
	"github.com/col3name/lines/pkg/common/domain"
	"github.com/col3name/lines/pkg/kiddy-line-processor/application/fake"
	"github.com/jackc/pgconn"
	"github.com/pashagolub/pgxmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

type inputStore struct {
	sport *domain.SportLine
}

type status int

const (
	failedStartTransaction status = iota
	failedDoRollback
	successDoRollback
	failedDoCommit
	successDoCommit

	tableNotExist
	failedQuery
	multipleTypes
	rowsError

	failedRowScan

	ok
)

type expectedStore struct {
	err    error
	result pgconn.CommandTag
	status status
}

func TestStore(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputStore
		expected *expectedStore
	}{
		{
			name: "failed start transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedStartTransaction,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "failed rollback transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedDoRollback,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "success rollback transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: successDoRollback,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "failed commit transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: failedDoCommit,
				err:    errors.ErrInternal,
				result: pgconn.CommandTag{},
			},
		},
		{
			name: "success commit transaction",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: successDoCommit,
				err:    nil,
				result: pgconn.CommandTag{},
			},
		},
		{
			name: "failed save",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: ok,
				err:    errors.ErrInternal,
				result: nil,
			},
		},
		{
			name: "success save",
			input: &inputStore{
				sport: &domain.SportLine{Type: domain.Baseball, Score: 0.744},
			},
			expected: &expectedStore{
				status: ok,
				err:    nil,
				result: pgxmock.NewResult("UPDATE", 1),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mock.Close()

			repo := NewSportLineRepository(mock, fake.Logger{})
			expected := test.expected
			expectedErr := expected.err
			switch expected.status {
			case ok:
				mock.ExpectBegin()
				exec := mock.ExpectExec("UPDATE sport_lines")
				if expectedErr != nil {
					exec.WillReturnError(expectedErr)
					mock.ExpectRollback()
				} else {
					exec.WillReturnResult(expected.result)
					mock.ExpectCommit().WillReturnError(nil)
				}
			case failedStartTransaction:
				mock.ExpectBegin().WillReturnError(expectedErr)
			case failedDoRollback:
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("UPDATE sport_lines").WillReturnError(expectedErr)
				mock.ExpectRollback().WillReturnError(expectedErr)
			case successDoRollback:
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("UPDATE sport_lines").
					WithArgs(test.input.sport.Score, test.input.sport.Type).
					WillReturnError(expectedErr)
				mock.ExpectRollback().WillReturnError(nil)
			case failedDoCommit:
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("UPDATE sport_lines").
					WithArgs(test.input.sport.Score, test.input.sport.Type).
					WillReturnResult(test.expected.result).
					WillReturnError(nil)
				mock.ExpectCommit().WillReturnError(expectedErr)
			case successDoCommit:
				mock.ExpectBegin().WillReturnError(nil)
				mock.ExpectExec("UPDATE sport_lines").
					WillReturnResult(expected.result).
					WillReturnError(nil)
				mock.ExpectCommit().WillReturnError(nil)
			}

			err = repo.Store(test.input.sport)
			assert.Equal(t, err, err)
			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

type inputGetLineBySport struct {
	sportTypes []domain.SportType
	status     status
	queryErr   error
}

type expectedGetLineBySport struct {
	lines []*domain.SportLine
	err   error
}

func TestGetSportLines(t *testing.T) {
	tests := []struct {
		name     string
		input    *inputGetLineBySport
		expected *expectedGetLineBySport
	}{
		{
			name:  "empty sport types list",
			input: &inputGetLineBySport{sportTypes: []domain.SportType{}},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInvalidArgument,
			},
		},
		{
			name: "table not exist error",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     tableNotExist,
				queryErr:   errBase.New("ERROR: relation \"sport_lines\" does not exist (SQLSTATE 42P01)"),
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrTableNotExist,
			},
		},
		{
			name: "failed query",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     failedQuery,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "multiple query",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball, domain.Soccer},
				status:     multipleTypes,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "return rows with error",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     rowsError,
				queryErr:   errors.ErrInternal,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "failed row scan",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     failedRowScan,
				queryErr:   nil,
			},
			expected: &expectedGetLineBySport{
				lines: nil,
				err:   errors.ErrInternal,
			},
		},
		{
			name: "success get lines",
			input: &inputGetLineBySport{
				sportTypes: []domain.SportType{domain.Baseball},
				status:     ok,
				queryErr:   nil,
			},
			expected: &expectedGetLineBySport{
				lines: []*domain.SportLine{
					{Type: domain.Baseball, Score: 0.744},
				},
				err: nil,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Errorf("an error '%s' was not expected when opening a stub database connection", err)
			}
			defer mock.Close()

			repo := NewSportLineRepository(mock, fake.Logger{})

			input := test.input
			expected := test.expected
			var data []interface{}
			for _, sportType := range input.sportTypes {
				data = append(data, sportType)
			}
			switch input.status {
			case tableNotExist:
				mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").WithArgs(data...).
					WillReturnError(input.queryErr)
			case failedQuery:
				mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").WithArgs(data...).
					WillReturnError(input.queryErr)
			case rowsError:
				r := pgxmock.NewRows([]string{"exists"}).AddRow(&domain.SportLine{
					Type:  domain.Baseball,
					Score: 0.744,
				})
				r.RowError(0, errors.ErrInternal)
				mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
					WillReturnError(nil).
					WillReturnRows(r.CloseError(errors.ErrInternal))
			case failedRowScan:
				rs := pgxmock.NewRows([]string{"type"})
				mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
					WillReturnError(nil).
					WillReturnRows(rs.AddRow("line.Score"))
			case multipleTypes:
				var args []interface{}
				for _, sportType := range input.sportTypes {
					args = append(args, sportType)
				}
				sql := "SELECT score,sport_type FROM sport_lines WHERE sport_type = (.+) UNION ALL SELECT score,sport_type FROM sport_lines WHERE sport_type =(.+);"
				mock.ExpectQuery(sql).
					WithArgs(args...)
			case ok:
				rs := pgxmock.NewRows([]string{"score", "type"})
				for _, line := range expected.lines {
					rs.AddRow(line.Score, line.Type)
				}
				mock.ExpectQuery("SELECT score,sport_type FROM sport_lines").
					WillReturnError(nil).
					WillReturnRows(rs)
			}

			types, err := repo.GetLinesBySportTypes(input.sportTypes)

			assert.Equal(t, expected.err, err)
			if expected.lines != nil {
				assert.Equal(t, len(expected.lines), len(types))
			} else {
				assert.Nil(t, types)
			}
			for i, expectedLine := range expected.lines {
				actualLine := types[i]
				assert.Equal(t, expectedLine.Type, actualLine.Type)
				assert.Equal(t, expectedLine.Score, actualLine.Score)
			}

			if err = mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}