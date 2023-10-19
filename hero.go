package main

import (
	"sync"
	"time"

	tcell "github.com/gdamore/tcell/v2"
)

const HeroEmoji rune = '\U0001F9B8' // 🦸‍♂️
const DeadEmoji rune = '\U0001F480' // 💀

type Hero struct {
	x              int
	y              int
	speed          uint8
	bombs          []*Bomb
	bombIdCounter  int
	isDead         bool
	enemies        []*Enemy
	enemyIdCounter int
	isPaused       bool
}

func (h *Hero) goRight(maxWidth int) {
	if h.isDead || h.isPaused {
		return
	}

	if h.x < maxWidth-2 {
		h.x += int(h.speed)
	}
}

func (h *Hero) goLeft() {
	if h.isDead || h.isPaused {
		return
	}

	if h.x > 0 {
		h.x -= int(h.speed)
	}
}

func (h *Hero) goUp() {
	if h.isDead || h.isPaused {
		return
	}

	if h.y > 0 {
		h.y -= int(h.speed)
	}
}

func (h *Hero) goDown(maxHeight int) {
	if h.isDead || h.isPaused {
		return
	}

	if h.y < maxHeight-1 {
		h.y += int(h.speed)
	}
}

func (h *Hero) draw(s tcell.Screen, style tcell.Style) {
	if h.isDead {
		s.SetContent(h.x, h.y, DeadEmoji, nil, style)
		return
	}

	s.SetContent(h.x, h.y, HeroEmoji, nil, style)

	for _, enemy := range h.enemies {
		if enemy.isDead {
			s.SetContent(enemy.currentX, enemy.currentY, DeadEmoji, nil, style)
			continue
		}

		s.SetContent(enemy.currentX, enemy.currentY, enemy.emoji, nil, style)
	}
}

func (h *Hero) addBomb(s tcell.Screen, style tcell.Style, clickedX, clickedY int) {
	if len(h.bombs) >= 5 || h.isDead || h.isPaused {
		return
	}

	finalX, finalY := clickedX, clickedY

	bomb := Bomb{
		bombId:    h.bombIdCounter,
		currentX:  h.x,
		currentY:  h.y,
		distanceX: makePositive(finalX - h.x),
		distanceY: makePositive(finalY - h.y),
		direction: calculateDirection(clickedX-h.x, clickedY-h.y),
		speed:     1,
		initialX:  h.x,
		initialY:  h.y,
		finalX:    finalX,
		finalY:    finalY,
		explodeIn: time.Second * 1,
		isDead:    false,
		hero:      h,
	}

	h.bombIdCounter++
	h.bombs = append(h.bombs, &bomb)

	ticker := time.NewTicker(20 * time.Millisecond)
	bombExploded := make(chan string)
	explosionComplete := make(chan struct{})

	var explosionWaitGroup sync.WaitGroup
	explosionWaitGroup.Add(2)

	bomb.draw(s, style, ticker, bombExploded, explosionComplete, &explosionWaitGroup)

	go func() {
		for {
			<-bombExploded

			explodedBomb := h.bombs[0]
			h.killTheThingsInTheExplosionArea(s, style, explodedBomb.currentX, explodedBomb.currentY)
			explodedBomb.isDead = true
			h.bombs = h.bombs[1:]

			explosionWaitGroup.Done()
			explosionWaitGroup.Wait()
		}
	}()

}

func (h *Hero) isNearBomb(bombX, bombY, actorX, actorY, distance int) bool {
	distanceX := makePositive(bombX - actorX)
	distanceY := makePositive(bombY - actorY)

	if distanceX <= distance && distanceY <= distance {
		return true
	}

	return false
}

func (h *Hero) removeIndexFromEnemySlice(idx int) {
	for i, enemy := range h.enemies {
		if enemy.id == idx {
			h.enemies = append(h.enemies[:i], h.enemies[i+1:]...)
			break
		}
	}
}

func (h *Hero) killTheThingsInTheExplosionArea(s tcell.Screen, style tcell.Style, bombX, bombY int) {
	if h.isNearBomb(bombX, bombY, h.x, h.y, 2) {
		h.isDead = true
		s.SetContent(h.x, h.y, DeadEmoji, nil, style)
		s.Show()
	}

	indexToRemove := []int{}
	for _, enemy := range h.enemies {
		if h.isNearBomb(bombX, bombY, enemy.currentX, enemy.currentY, 5) {
			enemy.isDead = true
			indexToRemove = append(indexToRemove, enemy.id)
			s.SetContent(enemy.currentX, enemy.currentY, DeadEmoji, nil, style)
			s.Show()
		}
	}

	for i := 0; i < len(indexToRemove); i++ {
		h.removeIndexFromEnemySlice(indexToRemove[i])
	}
}

func (h *Hero) spanNewEnemies(s tcell.Screen, style tcell.Style, maxWidth, maxHeight int) {
	if h.isDead {
		return
	}

	enemySpanTicker := time.NewTicker(2 * time.Second)

	go func() {
		for {
			<-enemySpanTicker.C
			if (len(h.enemies)) >= h.getEnemyQuantity() {
				continue
			}
			enemy := *NewEnemy(h.enemyIdCounter, maxWidth, maxHeight, h)
			newEnemySpeedTicker := time.NewTicker(time.Duration(h.getGameSpeed()) * time.Millisecond)
			h.enemyIdCounter++
			h.enemies = append(h.enemies, &enemy)
			enemy.draw(s, style, newEnemySpeedTicker)
		}
	}()
}

func (h *Hero) getGameSpeed() int {
	if h.enemyIdCounter < 51 {
		gameSpeed := 100 - int(h.enemyIdCounter)
		return gameSpeed
	}

	return 50
}

func (h *Hero) getEnemyQuantity() int {
	extraEnemies := h.enemyIdCounter / 25
	return extraEnemies + 2
}
