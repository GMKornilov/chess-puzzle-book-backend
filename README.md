# Chess puzzle generators

Provides methods for generating chess puzzles from 
pgn.

Provided methods:
- ```AnalyzeGame(path string, r io.Reader) ([]Task, error)``` - creates puzzles from one game from pgn
- ```AnalyzeAllGames(path string, r io.Reader) ([]Task, error)``` - analyzes all games from all games in pgn