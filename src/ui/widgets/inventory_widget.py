import json
import os
from PySide6.QtWidgets import (
    QWidget,
    QVBoxLayout,
    QHBoxLayout,
    QListWidget,
    QLineEdit,
    QPushButton,
    QLabel,
    QTabWidget,
)


class InventoryWidget(QWidget):
    def __init__(self):
        super().__init__()
        self.db = {}
        self._load_db()
        self._setup_ui()

    def _load_db(self):
        db_path = "db"
        for category in ["items", "weapons", "armors", "talismans"]:
            path = os.path.join(db_path, f"{category}.json")
            if os.path.exists(path):
                with open(path, "r", encoding="utf-8") as f:
                    self.db[category] = json.load(f)

    def _setup_ui(self):
        layout = QVBoxLayout(self)

        # Search Area
        search_layout = QHBoxLayout()
        self.txt_search = QLineEdit()
        self.txt_search.setPlaceholderText("Search for an item...")
        self.txt_search.textChanged.connect(self._on_search)
        search_layout.addWidget(QLabel("Search:"))
        search_layout.addWidget(self.txt_search)
        layout.addLayout(search_layout)

        # Tabs for categories
        self.tabs = QTabWidget()
        self.category_lists = {}

        for category in ["items", "weapons", "armors", "talismans"]:
            list_widget = QListWidget()
            self.category_lists[category] = list_widget
            self.tabs.addTab(list_widget, category.capitalize())
            self._populate_list(category)

        layout.addWidget(self.tabs)

        # Actions
        self.btn_add = QPushButton("Add Selected Item")
        layout.addWidget(self.btn_add)

    def _populate_list(self, category, filter_text=""):
        list_widget = self.category_lists[category]
        list_widget.clear()
        items = self.db.get(category, {})

        for item_id, name in items.items():
            if filter_text.lower() in name.lower():
                list_widget.addItem(f"{name} (ID: {item_id})")

    def _on_search(self, text):
        current_cat = self.tabs.tabText(self.tabs.currentIndex()).lower()
        self._populate_list(current_cat, text)

    def _add_item(self):
        # Placeholder for adding item logic
        current_cat = self.tabs.tabText(self.tabs.currentIndex()).lower()
        selected = self.category_lists[current_cat].currentItem()
        if selected:
            print(f"Adding item: {selected.text()}")
            # This will call SaveManager.add_item() in the future
