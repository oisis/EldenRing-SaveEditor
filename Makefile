.PHONY: setup install run build clean lint format

PYTHON = python3
VENV = .venv
BIN = $(VENV)/bin

setup:
	$(PYTHON) -m venv $(VENV)
	$(BIN)/pip install --upgrade pip
	$(BIN)/pip install -r requirements.txt

install:
	$(BIN)/pip install -r requirements.txt

run:
	$(BIN)/python src/main.py

build:
	$(BIN)/pyinstaller --clean --noconfirm --onefile --windowed \
		--add-data "db/*.json:db" \
		--add-data "src/ui/styles.qss:src/ui" \
		--name "EldenRingSaveEditor" \
		src/main.py

lint:
	$(BIN)/ruff check .

format:
	$(BIN)/ruff format .

clean:
	rm -rf build/ dist/ *.spec $(VENV) __pycache__ .ruff_cache
