import os
from PySide6.QtWidgets import (QMainWindow, QWidget, QHBoxLayout, QVBoxLayout, 
                             QListWidget, QStackedWidget, QLabel, QPushButton, 
                             QFileDialog, QMessageBox)
from PySide6.QtCore import Qt
from core.save_manager import SaveManager

class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("Elden Ring Save Editor")
        self.setMinimumSize(1000, 700)
        self.save_manager = None
        
        self._setup_ui()
        self._load_styles()

    def _setup_ui(self):
        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        main_layout = QHBoxLayout(central_widget)
        main_layout.setContentsMargins(0, 0, 0, 0)
        main_layout.setSpacing(0)

        # Sidebar
        self.sidebar = QListWidget()
        self.sidebar.setFixedWidth(200)
        self.sidebar.addItems(["General", "Inventory", "World Progress", "Character Slots"])
        self.sidebar.currentRowChanged.connect(self._on_sidebar_changed)
        main_layout.addWidget(self.sidebar)

        # Main Content Area
        content_container = QWidget()
        self.content_layout = QVBoxLayout(content_container)
        
        # Toolbar (Top)
        toolbar = QHBoxLayout()
        self.btn_open = QPushButton("Open Save")
        self.btn_open.clicked.connect(self._open_file)
        self.btn_save = QPushButton("Save Changes")
        self.btn_save.setEnabled(False)
        self.btn_save.clicked.connect(self._save_file)
        
        toolbar.addWidget(self.btn_open)
        toolbar.addWidget(self.btn_save)
        toolbar.addStretch()
        self.lbl_status = QLabel("No file loaded")
        toolbar.addWidget(self.lbl_status)
        
        self.content_layout.addLayout(toolbar)

        # Stacked Widget for pages
        self.pages = QStackedWidget()
        self._setup_pages()
        self.content_layout.addWidget(self.pages)

        main_layout.addWidget(content_container)

    def _setup_pages(self):
        # Page 0: General
        self.page_general = QWidget()
        layout = QVBoxLayout(self.page_general)
        layout.addWidget(QLabel("General Statistics (Placeholder)"))
        self.pages.addWidget(self.page_general)

        # Page 1: Inventory
        self.page_inventory = QWidget()
        layout = QVBoxLayout(self.page_inventory)
        layout.addWidget(QLabel("Inventory Editor (Placeholder)"))
        self.pages.addWidget(self.page_inventory)

        # Page 2: World Progress
        self.page_world = QWidget()
        layout = QVBoxLayout(self.page_world)
        layout.addWidget(QLabel("World Progress Editor (Placeholder)"))
        self.pages.addWidget(self.page_world)

        # Page 3: Character Slots
        self.page_slots = QWidget()
        self.slots_layout = QVBoxLayout(self.page_slots)
        self.slots_layout.addWidget(QLabel("Character Slot Management"))
        self.pages.addWidget(self.page_slots)

    def _load_styles(self):
        style_path = os.path.join(os.path.dirname(__file__), "styles.qss")
        if os.path.exists(style_path):
            with open(style_path, "r") as f:
                self.setStyleSheet(f.read())

    def _on_sidebar_changed(self, index):
        self.pages.setCurrentIndex(index)

    def _open_file(self):
        file_path, _ = QFileDialog.getOpenFileName(
            self, "Open Elden Ring Save", "", "Save Files (*.sl2 *.txt);;All Files (*)"
        )
        if file_path:
            try:
                self.save_manager = SaveManager(file_path)
                self.save_manager.load()
                self.lbl_status.setText(f"Loaded: {os.path.basename(file_path)}")
                self.btn_save.setEnabled(True)
                self._refresh_slots_ui()
                QMessageBox.information(self, "Success", f"Loaded {len(self.save_manager.slots)} potential slots.")
            except Exception as e:
                QMessageBox.critical(self, "Error", f"Failed to load save: {str(e)}")

    def _refresh_slots_ui(self):
        # Clear previous slots
        for i in reversed(range(self.slots_layout.count())): 
            widget = self.slots_layout.itemAt(i).widget()
            if widget and isinstance(widget, QPushButton):
                widget.setParent(None)
        
        # Add slot buttons
        if self.save_manager:
            for slot in self.save_manager.slots:
                status = "Active" if slot['active'] else "Empty"
                btn = QPushButton(f"Slot {slot['id']}: {slot['name']} ({status})")
                self.slots_layout.addWidget(btn)

    def _save_file(self):
        if self.save_manager:
            try:
                self.save_manager.save()
                QMessageBox.information(self, "Success", "Save file updated and backup created.")
            except Exception as e:
                QMessageBox.critical(self, "Error", f"Failed to save: {str(e)}")

if __name__ == "__main__":
    import sys
    from PySide6.QtWidgets import QApplication
    app = QApplication(sys.argv)
    window = MainWindow()
    window.show()
    sys.exit(app.exec())
