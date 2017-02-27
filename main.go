package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
)

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

type user struct {
	conn net.Conn
	out  chan string
}

func (u user) writePump() {
	for m := range u.out {
		u.conn.Write([]byte(m))
	}
}

type game struct {
	owner    *user
	enemy    *user
	num      int
	ownerNum int
	enemyNum int
}

type state struct {
	opts  chan func(*state)
	users map[*user]interface{}
	games map[*game]interface{}
	wait  *user
}

func (s *state) run() {
	for o := range s.opts {
		o(s)
	}
}

func main() {
	s := state{
		opts:  make(chan func(*state)),
		users: make(map[*user]interface{}),
		games: make(map[*game]interface{}),
	}
	go s.run()

	ln, err := net.Listen("tcp", "0.0.0.0:2007")
	if err != nil {
		panic(err)
	}

	for {
		conn, _ := ln.Accept()
		go func() {
			user := &user{
				conn: conn,
				out:  make(chan string),
			}
			go user.writePump()
			s.opts <- func(s *state) {
				s.users[user] = struct{}{}
				println("Connected")
			}
			s.opts <- func(s *state) {
				for u := range s.users {
					u.out <- "Joined new user\n"
				}
			}
			defer func() {
				s.opts <- func(s *state) {
					if s.wait == user {
						s.wait = nil
					}
					for g := range s.games {
						if g.owner == user {
							g.enemy.out <- "Enemy leave the game\n"
							delete(s.games, g)
							break
						}
						if g.enemy == user {
							g.owner.out <- "Enemy leave the game\n"
							delete(s.games, g)
							break
						}
					}
					delete(s.users, user)
					println("Disconnected")
				}
			}()
			sc := bufio.NewScanner(conn)
			for sc.Scan() {
				s.opts <- func(s *state) {
					text := sc.Text()
					switch text {
					case "ready":
						if s.wait != nil {
							g := &game{
								owner:    s.wait,
								enemy:    user,
								enemyNum: -1,
								ownerNum: -1,
								num:      rand.Intn(21),
							}
							s.games[g] = struct{}{}
							msg := "Game started!\nI guessed the number from [0 to 20], try to guess it, Who will close wins!\n"
							s.wait.out <- msg
							user.out <- msg
							s.wait = nil
						} else {
							s.wait = user
							user.out <- "please, wait for an another player\n"
						}
						break
					default:
						var currentGame *game = nil
						for g := range s.games {
							if g.owner == user || g.enemy == user {
								currentGame = g
								break
							}
						}
						if currentGame != nil {
							num, err := strconv.Atoi(text)
							if err != nil {
								user.out <- "It's not a num!\n"
								break
							}
							if num < 0 || num > 20 {
								user.out <- "Must be in range [0, 20]!\n"
								break
							}
							if currentGame.owner == user {
								if currentGame.ownerNum == -1 {
									currentGame.ownerNum = num
									if currentGame.enemyNum == -1 {
										user.out <- "Ok, wait another player...\n"
									}
								} else {
									user.out <- "You make your choice already!\n"
								}
							} else if currentGame.enemy == user {
								if currentGame.enemyNum == -1 {
									currentGame.enemyNum = num
									if currentGame.ownerNum == -1 {
										user.out <- "Ok, wait another player...\n"
									}
								} else {
									user.out <- "You make your choice already!\n"
								}
							}

							if currentGame != nil {
								g := currentGame
								if g.ownerNum != -1 && g.enemyNum != -1 {
									do := abs(g.ownerNum - g.num)
									de := abs(g.enemyNum - g.num)
									if do < de {
										g.owner.out <- fmt.Sprintf("You win the game, guessed num: %d, enemy num: %d!\n", g.num, g.enemyNum)
										g.enemy.out <- fmt.Sprintf("You loss the game, guessed num: %d, enemy num: %d!\n", g.num, g.ownerNum)
									} else if de < do {
										g.enemy.out <- fmt.Sprintf("You win the game, guessed num: %d, enemy num: %d!\n", g.num, g.ownerNum)
										g.owner.out <- fmt.Sprintf("You loss the game, guessed num: %d, enemy num: %d!\n", g.num, g.enemyNum)
									} else {
										g.enemy.out <- fmt.Sprintf("Draw, guessed num: %d, enemy num: %d!\n", g.num, g.ownerNum)
										g.owner.out <- fmt.Sprintf("Draw, guessed num: %d, enemy num: %d!\n", g.num, g.enemyNum)
									}
									delete(s.games, g)
								}
								break
							}
						}
						user.conn.Close()
						break
					}
				}
			}
		}()
	}
	fmt.Println("vim-go")
}
