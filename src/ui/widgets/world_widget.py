import json
import os
from PySide6.QtWidgets import (QWidget, QVBoxLayout, QHBoxLayout, QListWidget, 
                             QLineEdit, QPushButton, QLabel, QGroupBox, QTabWidget,
                             QCheckBox, QScrollArea)
from PySide6.QtCore import Qt, Signal

class WorldWidget(QWidget):
    progress_changed = Signal()

    def __init__(self):
        super().__init__()
        self.db = {}
        self._load_db()
        self._setup_ui()

    def _load_db(self):
        db_path = "db"
        for category in ["graces", "bosses"]:
            path = os.path.join(db_path, f"{category}.json")
            if os.path.exists(path):
                with open(path, "r", encoding="utf-8") as f:
                    self.db[category] = json.load(f)

    def _setup_ui(self):
        layout = QVBoxLayout(self)
        
        self.tabs = QTabWidget()
        self.checkboxes = {"graces": {}, "bosses": {}}
        
        for category in ["graces", "bosses"]:
            scroll = QScrollArea()
            scroll.setWidgetResizable(True)
            container = QWidget()
            vbox = QVBoxLayout(container)
            
            items = self.db.get(category, {})
            # Sort by name for easier navigation
            sorted_items = sorted(items.items(), key=lambda x: x[0])
            
            for enum_name, data in sorted_items:
                display_name = data.get("name", enum_name)
                cb = QCheckBox(display_name)
                cb.setProperty("id", data.get("id"))
                cb.stateChanged.connect(lambda: self.progress_changed.emit())
                vbox.addWidget(cb)
                self.checkboxes[category][data.get("id")] = cb
                
            vbox.addStretch()
            scroll.setWidget(container)
            self.tabs.addTab(scroll, category.capitalize())
            
        layout.addWidget(self.tabs)

    def load_progress(self, active_ids):
        """Sets checkboxes based on provided active IDs."""
        for category in self.checkboxes:
            for item_id, cb in self.checkboxes[category].items():
                cb.setChecked(item_id in active_ids)

    def get_selected_ids(self, category):
        """Returns a list of IDs that are checked in a given category."""
        selected = []
        for item_id, cb in self.checkboxes.get(category, {}).items():
            if cb.isChecked():
                selected.append(item_id)
        return selected
