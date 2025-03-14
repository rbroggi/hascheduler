package internal

import (
	"encoding/json"
	"net/http"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.store.FindSchedules(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// write schedules to response
	p, err := json.Marshal(schedules)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(p)
}

func (s *Service) Create(w http.ResponseWriter, r *http.Request) {
	// unmarshal request body into schedule
	var schedule Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// create schedule
	if err := s.store.CreateSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	// write schedule to response
	p, err := json.Marshal(schedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(p)
}

func (s *Service) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// unmarshal request body into schedule
	var schedule Schedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// update schedule
	schedule.ID = id
	if err := s.store.UpdateSchedule(r.Context(), &schedule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// write schedule to response
	p, err := json.Marshal(schedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(p)
}

func (s *Service) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// delete schedule
	schedule, err := s.store.DeleteSchedule(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	// write schedule to response
	p, err := json.Marshal(schedule)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(p)
}
