import os
from PySide6.QtWidgets import (QMainWindow, QWidget, QHBoxLayout, QVBoxLayout, 
                             QListWidget, QListWidgetItem, QStackedWidget, QLabel, 
                             QPushButton, QFileDialog, QMessageBox, QFrame)
from PySide6.QtGui import QIcon
from core.save_manager import SaveManager
from ui.widgets.stats_widget import StatsWidget
from ui.widgets.inventory_widget import InventoryWidget
from ui.widgets.world_widget import WorldWidget
from ui.widgets.importer_dialog import ImporterDialog
from utils.resource_manager import get_resource_path

class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle("Elden Ring Save Editor")
        self.setMinimumSize(1100, 800)
        
        # Set Window Icon (if exists)
        icon_path = get_resource_path("assets/32.png")
        if os.path.exists(icon_path):
            self.setWindowIcon(QIcon(icon_path))
            
        self.save_manager = None
        self.current_slot_id = None
        self.is_dark_mode = True
        
        self._setup_ui()
        self._apply_theme()

    def _setup_ui(self):
        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        main_layout = QHBoxLayout(central_widget)
        main_layout.setContentsMargins(10, 10, 10, 10)
        main_layout.setSpacing(15)

        # Sidebar
        sidebar_container = QWidget()
        sidebar_layout = QVBoxLayout(sidebar_container)
        sidebar_layout.setContentsMargins(0, 0, 0, 0)
        
        self.sidebar = QListWidget()
        self.sidebar.setFixedWidth(220)
        self._add_sidebar_item("General Stats", "contact-new")
        self.sidebar.addItem(QListWidgetItem(QIcon.fromTheme("contact-new"), "General Stats"))
        self.sidebar.addItem(QListWidgetItem(QIcon.fromTheme("package-x-generic"), "Inventory Editor"))
        self.sidebar.addItem(QListWidgetItem(QIcon.fromTheme("applications-internet"), "World Progress"))
        self.sidebar.addItem(QListWidgetItem(QIcon.fromTheme("system-users"), "Character Slots"))
        
        self.sidebar.currentRowChanged.connect(self._on_sidebar_changed)
        sidebar_layout.addWidget(self.sidebar)
        
        self.btn_theme = QPushButton(QIcon.fromTheme("display"), " Switch Theme")
        self.btn_theme.clicked.connect(self._toggle_theme)
        sidebar_layout.addWidget(self.btn_theme)
        
        main_layout.addWidget(sidebar_container)

        # Content Area
        content_container = QWidget()
        self.content_layout = QVBoxLayout(content_container)
        self.content_layout.setContentsMargins(0, 0, 0, 0)
        
        # Toolbar
        toolbar = QFrame()
        toolbar.setFrameShape(QFrame.StyledPanel)
        toolbar_layout = QHBoxLayout(toolbar)
        
        self.btn_open = QPushButton(QIcon.fromTheme("document-open"), " Open Save File")
        self.btn_open.clicked.connect(self._open_file)
        self.btn_save = QPushButton(QIcon.fromTheme("document-save"), " Save Changes")
        self.btn_save.setEnabled(False)
        self.btn_save.clicked.connect(self._save_file)
        
        toolbar_layout.addWidget(self.btn_open)
        toolbar_layout.addWidget(self.btn_save)
        toolbar_layout.addStretch()
        
        self.lbl_status = QLabel("Ready")
        self.lbl_status.setStyleSheet("font-weight: bold;")
        toolbar_layout.addWidget(self.lbl_status)
        
        self.content_layout.addWidget(toolbar)

        # Pages
        self.pages = QStackedWidget()
        self._setup_pages()
        self.content_layout.addWidget(self.pages)

        main_layout.addWidget(content_container)

    def _add_sidebar_item(self, text, icon_name):
        # Helper to clear previous manual items if needed
        pass

    def _setup_pages(self):
        self.stats_widget = StatsWidget()
        self.stats_widget.stats_changed.connect(self._on_stats_modified)
        self.pages.addWidget(self.stats_widget)

        self.inventory_widget = InventoryWidget()
        self.inventory_widget.btn_add.setIcon(QIcon.fromTheme("list-add"))
        self.inventory_widget.btn_add.clicked.connect(self._on_add_item)
        self.pages.addWidget(self.inventory_widget)

        self.world_widget = WorldWidget()
        self.world_widget.progress_changed.connect(self._on_progress_modified)
        self.pages.addWidget(self.world_widget)

        self.page_slots = QWidget()
        self.slots_layout = QVBoxLayout(self.page_slots)
        self.slots_layout.setSpacing(10)
        
        actions_layout = QHBoxLayout()
        self.btn_import = QPushButton(QIcon.fromTheme("edit-copy"), " Import Character")
        self.btn_import.clicked.connect(self._import_character)
        self.btn_delete = QPushButton(QIcon.fromTheme("edit-delete"), " Delete Character")
        self.btn_delete.clicked.connect(self._delete_character)
        self.btn_delete.setStyleSheet("background-color: #a83232;")
        
        actions_layout.addWidget(self.btn_import)
        actions_layout.addWidget(self.btn_delete)
        self.slots_layout.addLayout(actions_layout)
        
        header = QLabel("Select a character slot to edit:")
        header.setStyleSheet("font-size: 16px; font-weight: bold; margin-top: 10px;")
        self.slots_layout.addWidget(header)
        self.pages.addWidget(self.page_slots)

    def _toggle_theme(self):
        self.is_dark_mode = not self.is_dark_mode
        self.btn_theme.setText(" Switch to Dark Mode" if self.is_dark_mode else " Switch to Light Mode")
        self._apply_theme()

    def _apply_theme(self):
        theme_file = "src/ui/styles.qss" if self.is_dark_mode else "src/ui/light_style.qss"
        style_path = get_resource_path(theme_file)
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
                self._refresh_slots_ui()
                self.sidebar.setCurrentRow(3)
            except Exception as e:
                QMessageBox.critical(self, "Error", f"Failed to load save: {str(e)}")

    def _refresh_slots_ui(self):
        for i in reversed(range(self.slots_layout.count())): 
            item = self.slots_layout.itemAt(i)
            if item and item.widget() and isinstance(item.widget(), QPushButton) and item.widget() not in [self.btn_import, self.btn_delete]:
                item.widget().setParent(None)
        
        if self.save_manager:
            for slot in self.save_manager.slots:
                status = "Active" if slot['active'] else "Empty"
                icon = QIcon.fromTheme("user-available" if slot['active'] else "user-away")
                btn = QPushButton(icon, f" Slot {slot['id']}: {slot['name']} ({status})")
                btn.setMinimumHeight(50)
                btn.clicked.connect(lambda checked=False, s_id=slot['id']: self._select_slot(s_id))
                self.slots_layout.addWidget(btn)
            self.slots_layout.addStretch()

    def _select_slot(self, slot_id):
        self.current_slot_id = slot_id
        stats = self.save_manager.get_character_stats(slot_id)
        self.stats_widget.load_stats(stats)
        
        active_flags = []
        for category in ["graces", "bosses"]:
            items = self.world_widget.db.get(category, {})
            for data in items.values():
                flag_id = data.get("id")
                if self.save_manager.get_event_flag(slot_id, flag_id):
                    active_flags.append(flag_id)
        self.world_widget.load_progress(active_flags)
        
        self.sidebar.setCurrentRow(0)
        self.lbl_status.setText(f"Editing: {stats.name}")

    def _on_stats_modified(self):
        if self.save_manager and self.current_slot_id is not None:
            new_stats = self.stats_widget.get_stats()
            self.save_manager.update_character_stats(self.current_slot_id, new_stats)
            self.btn_save.setEnabled(True)

    def _on_progress_modified(self):
        if self.save_manager and self.current_slot_id is not None:
            for category in ["graces", "bosses"]:
                items = self.world_widget.db.get(category, {})
                selected_ids = self.world_widget.get_selected_ids(category)
                for data in items.values():
                    flag_id = data.get("id")
                    self.save_manager.set_event_flag(self.current_slot_id, flag_id, flag_id in selected_ids)
            self.btn_save.setEnabled(True)

    def _on_add_item(self):
        if self.save_manager and self.current_slot_id is not None:
            current_cat = self.inventory_widget.tabs.tabText(self.inventory_widget.tabs.currentIndex()).lower()
            selected = self.inventory_widget.category_lists[current_cat].currentItem()
            if selected:
                item_id = int(selected.text().split("ID: ")[1].strip(")"))
                if self.save_manager.add_item(self.current_slot_id, item_id):
                    self.btn_save.setEnabled(True)
                    QMessageBox.information(self, "Success", f"Item added to inventory.")

    def _delete_character(self):
        if not self.save_manager or self.current_slot_id is None:
            QMessageBox.warning(self, "Warning", "Please select a character slot first.")
            return
        
        slot_name = self.save_manager.slots[self.current_slot_id]["name"]
        reply = QMessageBox.question(self, "Confirm Delete", 
                                   f"Are you sure you want to delete character '{slot_name}'? This cannot be undone.",
                                   QMessageBox.Yes | QMessageBox.No)
        
        if reply == QMessageBox.Yes:
            if self.save_manager.delete_character(self.current_slot_id):
                self.btn_save.setEnabled(True)
                self._refresh_slots_ui()
                self.sidebar.setCurrentRow(3)
                QMessageBox.information(self, "Success", "Character deleted.")

    def _import_character(self):
        if not self.save_manager:
            return
        dialog = ImporterDialog(self)
        if dialog.exec():
            source_manager = dialog.source_manager
            source_slot_id = dialog.get_selected_slot_id()
            target_id = 9
            if self.save_manager.import_character(source_manager, source_slot_id, target_id):
                self.btn_save.setEnabled(True)
                self._refresh_slots_ui()
                QMessageBox.information(self, "Success", f"Character imported to Slot {target_id}.")

    def _save_file(self):
        if self.save_manager:
            try:
                self.save_manager.save()
                self.btn_save.setEnabled(False)
                QMessageBox.information(self, "Success", "Save file updated and backup created.")
                self._refresh_slots_ui()
            except Exception as e:
                QMessageBox.critical(self, "Error", f"Failed to save: {str(e)}")
