#!/bin/sh

session="go-zeebe"

attach() {
    [ -n "${TMUX:-}" ] &&
        tmux switch-client -t $session ||
        tmux attach-session -t $session
}

if tmux has-session -t "=$session" 2> /dev/null; then
    attach
    exit 0
fi


tmux new-session -d -s $session

# 1st divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 24
tmux select-pane -D

# 2nd divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 20
tmux select-pane -D

# 3rd divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 16
tmux select-pane -D

# 4th divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 12
tmux select-pane -D

# 5th divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 8
tmux select-pane -D

# 6th divided horizontal pane
tmux split-window -v
tmux select-pane -U
tmux split-window -h
tmux resize-pane -R 40
tmux resize-pane -U 4
tmux select-pane -D

# Run 6th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8006 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8006/stats | jq ."' Enter

# Run 5th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8005 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8005/stats | jq ."' Enter

# Run 4th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8004 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8004/stats | jq ."' Enter

# Run 3th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8003 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8003/stats | jq ."' Enter

# Run 2th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8002 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8002/stats | jq ."' Enter

# Run 1th worker and display stats
tmux select-pane -U
tmux select-pane -L
tmux send-keys -t $session 'SERVER_PORT=:8001 make worker' Enter
tmux select-pane -R
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8001/stats | jq ."' Enter

# Return to command pane
tmux select-pane -D
tmux select-pane -D
tmux select-pane -D
tmux select-pane -D
tmux select-pane -D
tmux select-pane -D

# Run stats and prepare for starter command
tmux split-window -h
tmux resize-pane -R 40
tmux send-keys -t $session 'watch -n1 "curl -sS localhost:8080/stats | jq ."' Enter
tmux select-pane -L

attach
