package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

type server struct {
	rooms    map[string]*room
	commands chan command
}

func newServer() *server {
	return &server{
		rooms:    make(map[string]*room),
		commands: make(chan command),
	}
}

func (s *server) newClient(conn net.Conn) {
	log.Printf("Client has connected: %s", conn.RemoteAddr().String())

	c := &client{
		conn:     conn,
		name:     "Anonymous",
		commands: s.commands,
	}

	c.readInput()
}

func (s *server) run() {
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_NAME:
			s.name(cmd.client, cmd.args)
		case CMD_JOIN:
			s.join(cmd.client, cmd.args)
		case CMD_ROOMS:
			s.listRooms(cmd.client, cmd.args)
		case CMD_MESSAGE:
			s.msg(cmd.client, cmd.args)
		case CMD_MEMBERS:
			s.members(cmd.client, cmd.args)
		case CMD_QUIT:
			s.quit(cmd.client, cmd.args)
		}
	}
}

func (s *server) name(c *client, args []string) {
	c.name = args[1]
	c.msg(fmt.Sprintf("Name is %s", c.name))
}

func (s *server) join(c *client, args []string) {
	roomName := args[1]
	r, ok := s.rooms[roomName]
	if !ok {
		r = &room{
			name:    roomName,
			members: make(map[net.Addr]*client),
		}
		s.rooms[roomName] = r
	}
	r.members[c.conn.RemoteAddr()] = c
	s.leaveCurrentRoom(c)
	c.room = r
	r.broadcast(c, fmt.Sprintf("%s has joined the room", c.name))
	c.msg(fmt.Sprintf("You have joined the room %s", c.name))
}

func (s *server) listRooms(c *client, args []string) {
	var rooms []string
	for name := range s.rooms {
		rooms = append(rooms, name)
	}
	c.msg(fmt.Sprintf("Open rooms: %s", strings.Join(rooms, ", ")))
}

func (s *server) msg(c *client, args []string) {
	if c.room == nil {
		c.err(errors.New("Not in a room"))
		return
	}
	c.room.broadcast(c, c.name+": "+strings.Join(args[1:len(args)], " "))
}

func (s *server) members(c *client, args []string) {
	if c.room == nil {
		c.err(errors.New("Not in a room"))
		return
	}
	c.msg(fmt.Sprintf("Members in room: %s", strings.Join(c.room.membersList(c), ", ")))
}

func (s *server) quit(c *client, args []string) {
	log.Printf("Client has disconnect: %s", c.conn.RemoteAddr().String())
	s.leaveCurrentRoom(c)
	c.msg("Closing connection")
	c.conn.Close()
}

func (s *server) leaveCurrentRoom(c *client) {
	if c.room != nil {
		delete(c.room.members, c.conn.RemoteAddr())
		c.room.broadcast(c, fmt.Sprintf("%s has left the room", c.name))
	}
}
