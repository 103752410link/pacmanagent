'use strict';
/*!
 * Pacman Agent Arena - Spectator Client
 */

(function() {
    // Canvas setup
    var canvas = document.getElementById('canvas');
    var ctx = canvas.getContext('2d');
    var W = canvas.width;
    var H = canvas.height;

    // Game state
    var gameState = null;
    var maze = null;
    var wallColor = '#09f';
    var connected = false;

    // Constants (matching game.go)
    var CELL_SIZE = 20;
    var MAP_OFFSET_X = 60;
    var MAP_OFFSET_Y = 10;
    var COS = [1, 0, -1, 0];
    var SIN = [0, 1, 0, -1];

    // Connect to WebSocket
    function connect() {
        var wsUrl = 'ws://' + window.location.host + '/ws';
        var ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            connected = true;
            updateInfo('已连接 - 等待游戏开始');
            // Fetch maze and initial state
            fetch('/api/game/maze').then(function(r) { return r.json(); }).then(function(data) {
                maze = data.maze;
                wallColor = data.wall_color;
            });
        };

        ws.onmessage = function(event) {
            try {
                gameState = JSON.parse(event.data);
            } catch(e) {}
        };

        ws.onclose = function() {
            connected = false;
            updateInfo('连接断开，重连中...');
            setTimeout(connect, 3000);
        };

        ws.onerror = function() {
            updateInfo('连接失败');
        };
    }

    function updateInfo(text) {
        var el = document.getElementById('game-info');
        if (el) {
            el.innerHTML = '<p>' + text + '</p>';
        }
    }

    // Fetch maze data from server
    function fetchMaze() {
        fetch('/api/game/state')
            .then(function(r) { return r.json(); })
            .then(function(state) {
                // We need a separate endpoint for maze data
            })
            .catch(function() {});
    }

    // Default maze (level 1)
    var DEFAULT_MAZE = [
        [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1],
        [1,0,0,0,0,0,0,0,0,0,0,0,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,1],
        [1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1],
        [1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1],
        [1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1],
        [1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1],
        [1,0,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,0,1],
        [1,0,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,0,1],
        [1,0,0,0,0,0,0,1,1,0,0,0,0,1,1,0,0,0,0,1,1,0,0,0,0,0,0,1],
        [1,1,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,0,0,0,0,0,0,0,0,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,1,1,1,2,2,1,1,1,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,1,2,2,2,2,2,2,1,0,1,1,0,1,1,1,1,1,1],
        [0,0,0,0,0,0,0,0,0,0,1,2,2,2,2,2,2,1,0,0,0,0,0,0,0,0,0,0],
        [1,1,1,1,1,1,0,1,1,0,1,2,2,2,2,2,2,1,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,0,0,0,0,0,0,0,0,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1],
        [1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1],
        [1,0,0,0,0,0,0,0,0,0,0,0,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,1],
        [1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1],
        [1,0,1,1,1,1,0,1,1,1,1,1,0,1,1,0,1,1,1,1,1,0,1,1,1,1,0,1],
        [1,0,0,0,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1,1,0,0,0,1],
        [1,1,1,0,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,0,1,1,1],
        [1,1,1,0,1,1,0,1,1,0,1,1,1,1,1,1,1,1,0,1,1,0,1,1,0,1,1,1],
        [1,0,0,0,0,0,0,1,1,0,0,0,0,1,1,0,0,0,0,1,1,0,0,0,0,0,0,1],
        [1,0,1,1,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,1,1,0,1],
        [1,0,1,1,1,1,1,1,1,1,1,1,0,1,1,0,1,1,1,1,1,1,1,1,1,1,0,1],
        [1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,1],
        [1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1,1]
    ];

    maze = DEFAULT_MAZE;

    // Draw functions
    function drawMaze() {
        if (!maze) return;

        ctx.lineWidth = 2;
        for (var j = 0; j < maze.length; j++) {
            for (var i = 0; i < maze[j].length; i++) {
                var value = maze[j][i];
                if (value) {
                    var code = [0, 0, 0, 0];
                    if (getMaze(i+1, j) && !(getMaze(i+1, j-1) && getMaze(i+1, j+1) && getMaze(i, j-1) && getMaze(i, j+1))) {
                        code[0] = 1;
                    }
                    if (getMaze(i, j+1) && !(getMaze(i-1, j+1) && getMaze(i+1, j+1) && getMaze(i-1, j) && getMaze(i+1, j))) {
                        code[1] = 1;
                    }
                    if (getMaze(i-1, j) && !(getMaze(i-1, j-1) && getMaze(i-1, j+1) && getMaze(i, j-1) && getMaze(i, j+1))) {
                        code[2] = 1;
                    }
                    if (getMaze(i, j-1) && !(getMaze(i-1, j-1) && getMaze(i+1, j-1) && getMaze(i-1, j) && getMaze(i+1, j))) {
                        code[3] = 1;
                    }
                    if (code.indexOf(1) > -1) {
                        ctx.strokeStyle = value == 2 ? '#FFF' : wallColor;
                        var pos = coord2Position(i, j);
                        switch (code.join('')) {
                            case '1100':
                                ctx.beginPath();
                                ctx.arc(pos.x + CELL_SIZE/2, pos.y + CELL_SIZE/2, CELL_SIZE/2, Math.PI, 1.5*Math.PI, false);
                                ctx.stroke();
                                ctx.closePath();
                                break;
                            case '0110':
                                ctx.beginPath();
                                ctx.arc(pos.x - CELL_SIZE/2, pos.y + CELL_SIZE/2, CELL_SIZE/2, 1.5*Math.PI, 2*Math.PI, false);
                                ctx.stroke();
                                ctx.closePath();
                                break;
                            case '0011':
                                ctx.beginPath();
                                ctx.arc(pos.x - CELL_SIZE/2, pos.y - CELL_SIZE/2, CELL_SIZE/2, 0, 0.5*Math.PI, false);
                                ctx.stroke();
                                ctx.closePath();
                                break;
                            case '1001':
                                ctx.beginPath();
                                ctx.arc(pos.x + CELL_SIZE/2, pos.y - CELL_SIZE/2, CELL_SIZE/2, 0.5*Math.PI, 1*Math.PI, false);
                                ctx.stroke();
                                ctx.closePath();
                                break;
                            default:
                                var dist = CELL_SIZE / 2;
                                code.forEach(function(v, index) {
                                    if (v) {
                                        ctx.beginPath();
                                        ctx.moveTo(pos.x, pos.y);
                                        ctx.lineTo(pos.x - COS[index] * dist, pos.y - SIN[index] * dist);
                                        ctx.stroke();
                                        ctx.closePath();
                                    }
                                });
                        }
                    }
                }
            }
        }
    }

    function getMaze(x, y) {
        if (maze && maze[y] && typeof maze[y][x] !== 'undefined') {
            return maze[y][x];
        }
        return -1;
    }

    function coord2Position(cx, cy) {
        return {
            x: MAP_OFFSET_X + cx * CELL_SIZE + CELL_SIZE / 2,
            y: MAP_OFFSET_Y + cy * CELL_SIZE + CELL_SIZE / 2
        };
    }

    function drawBeans(beans) {
        if (!beans) return;
        ctx.fillStyle = '#F5F5DC';
        beans.forEach(function(bean) {
            var pos = coord2Position(bean.X || bean.x, bean.Y || bean.y);
            ctx.fillRect(pos.x - 2, pos.y - 2, 4, 4);
        });
    }

    function drawPlayer(player) {
        if (!player || !player.alive) return;

        ctx.fillStyle = '#FFE600';
        ctx.beginPath();
        var x = MAP_OFFSET_X + player.x * CELL_SIZE + CELL_SIZE / 2;
        var y = MAP_OFFSET_Y + player.y * CELL_SIZE + CELL_SIZE / 2;
        var r = CELL_SIZE * 0.7;
        var angle = player.orientation * 0.5 * Math.PI;
        ctx.arc(x, y, r, (angle + 0.2) * Math.PI, (angle - 0.2) * Math.PI, false);
        ctx.lineTo(x, y);
        ctx.closePath();
        ctx.fill();
    }

    function drawGhosts(ghosts) {
        if (!ghosts) return;

        ghosts.forEach(function(ghost) {
            var x = MAP_OFFSET_X + ghost.x * CELL_SIZE + CELL_SIZE / 2;
            var y = MAP_OFFSET_Y + ghost.y * CELL_SIZE + CELL_SIZE / 2;
            var w = CELL_SIZE * 1.5;
            var h = CELL_SIZE * 1.5;

            var isSick = ghost.status === 3;
            ctx.fillStyle = isSick ? '#BABABA' : (ghost.color || '#F00');

            ctx.beginPath();
            ctx.arc(x, y, w * 0.5, 0, Math.PI, true);
            ctx.lineTo(x - w * 0.5, y + h * 0.4);
            ctx.quadraticCurveTo(x - w * 0.4, y + h * 0.5, x - w * 0.2, y + h * 0.3);
            ctx.quadraticCurveTo(x, y + h * 0.5, x + w * 0.2, y + h * 0.3);
            ctx.quadraticCurveTo(x + w * 0.4, y + h * 0.5, x + w * 0.5, y + h * 0.4);
            ctx.fill();
            ctx.closePath();

            // Eyes
            ctx.fillStyle = '#FFF';
            if (isSick) {
                ctx.beginPath();
                ctx.arc(x - w * 0.15, y - h * 0.21, w * 0.08, 0, 2 * Math.PI, false);
                ctx.arc(x + w * 0.15, y - h * 0.21, w * 0.08, 0, 2 * Math.PI, false);
                ctx.fill();
                ctx.closePath();
            } else {
                ctx.beginPath();
                ctx.arc(x - w * 0.15, y - h * 0.21, w * 0.12, 0, 2 * Math.PI, false);
                ctx.arc(x + w * 0.15, y - h * 0.21, w * 0.12, 0, 2 * Math.PI, false);
                ctx.fill();
                ctx.closePath();
                ctx.fillStyle = '#000';
                ctx.beginPath();
                ctx.arc(x - w * 0.12, y - h * 0.18, w * 0.07, 0, 2 * Math.PI, false);
                ctx.arc(x + w * 0.18, y - h * 0.18, w * 0.07, 0, 2 * Math.PI, false);
                ctx.fill();
                ctx.closePath();
            }

            // Name label
            ctx.font = '8px monospace';
            ctx.fillStyle = '#AAA';
            ctx.textAlign = 'center';
            ctx.fillText(ghost.name || ghost.id, x, y + h * 0.8);
        });
    }

    function drawHUD(state) {
        if (!state) return;

        var x = 720;

        // Background panel
        ctx.fillStyle = 'rgba(20, 20, 40, 0.9)';
        ctx.strokeStyle = '#333';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.roundRect(x - 15, 10, 240, 380, 8);
        ctx.fill();
        ctx.stroke();

        // Score
        ctx.textAlign = 'left';
        ctx.textBaseline = 'top';
        ctx.fillStyle = '#F88';
        ctx.font = '13px sans-serif';
        ctx.fillText('SCORE', x, 22);
        ctx.fillStyle = '#FFF';
        ctx.font = 'bold 32px sans-serif';
        ctx.fillText(String(state.score || 0), x, 40);

        // Divider
        ctx.strokeStyle = '#444';
        ctx.beginPath();
        ctx.moveTo(x, 80);
        ctx.lineTo(x + 200, 80);
        ctx.stroke();

        // Level
        ctx.fillStyle = '#88F';
        ctx.font = '13px sans-serif';
        ctx.fillText('LEVEL', x, 90);
        ctx.fillStyle = '#FFF';
        ctx.font = 'bold 24px sans-serif';
        ctx.fillText(String(state.level || 1), x, 108);

        // Divider
        ctx.strokeStyle = '#444';
        ctx.beginPath();
        ctx.moveTo(x, 140);
        ctx.lineTo(x + 200, 140);
        ctx.stroke();

        // Round
        ctx.fillStyle = '#8F8';
        ctx.font = '13px sans-serif';
        ctx.fillText('ROUND', x, 150);
        ctx.fillStyle = '#FFF';
        ctx.font = 'bold 24px sans-serif';
        ctx.fillText(String(state.round || 0), x, 168);

        // Phase
        ctx.fillStyle = '#AAA';
        ctx.font = '14px sans-serif';
        var phaseText = (state.phase === 'ghost_turn') ? 'GHOST TURN' : 'PACMAN TURN';
        ctx.fillText(phaseText, x, 205);

        // Divider
        ctx.strokeStyle = '#444';
        ctx.beginPath();
        ctx.moveTo(x, 230);
        ctx.lineTo(x + 200, 230);
        ctx.stroke();

        // Ghosts
        ctx.fillStyle = '#F55';
        ctx.font = '13px sans-serif';
        ctx.fillText('GHOSTS', x, 242);
        ctx.fillStyle = '#FFF';
        ctx.font = '16px sans-serif';
        var ghosts = state.ghosts || [];
        for (var i = 0; i < ghosts.length; i++) {
            ctx.fillStyle = ghosts[i].color || '#F00';
            ctx.fillText(ghosts[i].name || ghosts[i].id, x, 260 + i * 22);
        }

        // Beans
        ctx.fillStyle = '#AAA';
        ctx.font = '13px sans-serif';
        ctx.fillText('BEANS: ' + (state.beans_remaining || 0), x, 340);

        // Status
        if (state.paused) {
            ctx.font = '20px sans-serif';
            ctx.fillStyle = '#FF0';
            ctx.fillText('PAUSED', x, 360);
        }

        // Game over
        if (state.game_over) {
            ctx.font = 'bold 48px monospace';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillStyle = '#FFF';
            ctx.fillText(state.player && state.player.alive ? 'YOU WIN!' : 'GAME OVER', W / 2, H * 0.35);
            ctx.font = '24px sans-serif';
            ctx.fillText('FINAL SCORE: ' + (state.score || 0), W / 2, H * 0.5);
        }
    }

    // Main render loop
    function render() {
        ctx.fillStyle = '#000';
        ctx.fillRect(0, 0, W, H);

        drawMaze();

        if (gameState) {
            drawBeans(gameState.beans || []);
            drawPlayer(gameState.player);
            drawGhosts(gameState.ghosts);
            drawHUD(gameState);

            if (!connected) {
                ctx.font = '16px monospace';
                ctx.textAlign = 'center';
                ctx.fillStyle = '#F00';
                ctx.fillText('DISCONNECTED - Reconnecting...', W / 2, H - 30);
            }
        } else {
            ctx.font = '16px monospace';
            ctx.textAlign = 'center';
            ctx.fillStyle = '#AAA';
            ctx.fillText('Waiting for game state...', W / 2, H / 2);
        }

        requestAnimationFrame(render);
    }

    // Start
    connect();
    render();
})();
