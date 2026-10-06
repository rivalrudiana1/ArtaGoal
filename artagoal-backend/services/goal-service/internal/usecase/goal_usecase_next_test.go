package usecase

import (
	"context"
	"testing"
	"time"

	"artagoal/goal-service/internal/domain/entity"
	"artagoal/goal-service/internal/repository"
)

// stubRepo adalah fake in-memory untuk verifikasi logika usecase
// tanpa membutuhkan Postgres.
type stubRepo struct {
	repository.GoalRepository
	goals    map[string]entity.Goal
	contribs map[string]entity.GoalContribution
	byGoal   map[string][]entity.GoalContribution
}

func newStub() *stubRepo {
	return &stubRepo{goals: map[string]entity.Goal{}, contribs: map[string]entity.GoalContribution{}, byGoal: map[string][]entity.GoalContribution{}}
}

func (s *stubRepo) GetGoalByID(_ context.Context, id string) (entity.Goal, error) {
	g, ok := s.goals[id]
	if !ok {
		return entity.Goal{}, entity.ErrGoalNotFound
	}
	return g, nil
}

func (s *stubRepo) AddContribution(_ context.Context, goalID string, amount float64, note *string) (entity.GoalContribution, entity.Goal, error) {
	g := s.goals[goalID]
	c := entity.GoalContribution{ID: "c1", GoalID: goalID, Amount: amount, Note: note, CreatedAt: time.Now()}
	g.CurrentAmount += amount
	if g.CurrentAmount >= g.TargetAmount {
		g.Status = entity.GoalStatusActive
		// repo asli set achieved; tiru:
		g.Status = entity.GoalStatusAchieved
	}
	s.goals[goalID] = g
	s.contribs[c.ID] = c
	s.byGoal[goalID] = append(s.byGoal[goalID], c)
	return c, g, nil
}

func (s *stubRepo) ListContributionsByGoalID(_ context.Context, goalID string) ([]entity.GoalContribution, error) {
	if _, ok := s.goals[goalID]; !ok {
		return nil, entity.ErrGoalNotFound
	}
	return s.byGoal[goalID], nil
}

func (s *stubRepo) DeleteContribution(_ context.Context, id string) (entity.Goal, error) {
	c, ok := s.contribs[id]
	if !ok {
		return entity.Goal{}, entity.ErrContributionNotFound
	}
	g := s.goals[c.GoalID]
	g.CurrentAmount -= c.Amount
	if g.CurrentAmount < 0 {
		g.CurrentAmount = 0
	}
	if g.Status == entity.GoalStatusAchieved && g.CurrentAmount < g.TargetAmount {
		g.Status = entity.GoalStatusActive
	}
	s.goals[c.GoalID] = g
	delete(s.contribs, id)
	return g, nil
}

func TestAddContributionValidation(t *testing.T) {
	st := newStub()
	id := "g1"
	st.goals[id] = entity.Goal{ID: id, UserID: "u1", Title: "DP Rumah", TargetAmount: 1000, Status: entity.GoalStatusActive}
	uc := NewGoalUsecase(st)
	ctx := context.Background()

	if _, _, err := uc.AddContribution(ctx, id, 0, nil); err == nil {
		t.Fatal("amount 0 harus ditolak")
	}
	if _, _, err := uc.AddContribution(ctx, id, -5, nil); err == nil {
		t.Fatal("amount negatif harus ditolak")
	}
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	s := string(long)
	if _, _, err := uc.AddContribution(ctx, id, 100, &s); err == nil {
		t.Fatal("note >500 harus ditolak")
	}
	c, g, err := uc.AddContribution(ctx, id, 1200, nil)
	if err != nil {
		t.Fatalf("happy path gagal: %v", err)
	}
	if c.Amount != 1200 || g.CurrentAmount != 1200 {
		t.Fatalf("saldo salah: %+v %+v", c, g)
	}
	if g.Status != entity.GoalStatusAchieved {
		t.Fatalf("harus auto-achieved, dapat %q", g.Status)
	}
}

func TestAddContributionCancelledRejected(t *testing.T) {
	st := newStub()
	st.goals["g9"] = entity.Goal{ID: "g9", UserID: "u1", Title: "X", TargetAmount: 100, Status: entity.GoalStatusCancelled}
	uc := NewGoalUsecase(st)
	if _, _, err := uc.AddContribution(context.Background(), "g9", 10, nil); err == nil {
		t.Fatal("goal cancelled harus ditolak")
	}
}

func TestGoalProgressMath(t *testing.T) {
	st := newStub()
	uc := NewGoalUsecase(st)
	td := time.Now().AddDate(1, 0, 0)
	g := entity.Goal{TargetAmount: 1200, CurrentAmount: 300, ExpectedInflationRate: 0, TargetDate: &td, Status: "active"}
	// akses via interface: GoalProgress ada di interface
	p := uc.GoalProgress(g)
	if p.PercentComplete != 25 {
		t.Fatalf("percent harus 25, dapat %v", p.PercentComplete)
	}
	if p.RemainingAmount != 900 {
		t.Fatalf("remaining harus 900, dapat %v", p.RemainingAmount)
	}
	if p.MonthsRemaining <= 0 {
		t.Fatalf("months remaining harus >0, dapat %v", p.MonthsRemaining)
	}
}

