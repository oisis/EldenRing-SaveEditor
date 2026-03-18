import os
from PySide6.QtWidgets import (QMainWindow, QWidget, QHBoxLayout, QVBoxLayout, 
                             QListWidget, QStackedWidget, QLabel, QPushButton, 
                             QFileDialog, QMessageBox)
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
        self.setMinimumSize(1000, 700)
        self.save_manager = None
        self.current_slot_id = None
        
        self._setup_ui()
        self._load_styles()

    def _setup_ui(self):
        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        main_layout = QHBoxLayout(central_widget)
        main_layout.setContentsMargins(0, 0, 0, 0)
        main_layout.setSpacing(0)

        self.sidebar = QListWidget()
        self.sidebar.setFixedWidth(200)
        self.sidebar.addItems(["General", "Inventory", "World Progress", "Character Slots"])
        self.sidebar.currentRowChanged.connect(self._on_sidebar_changed)
        main_layout.addWidget(self.sidebar)

        content_container = QWidget()
        self.content_layout = QVBoxLayout(content_container)
        
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

        self.pages = QStackedWidget()
        self._setup_pages()
        self.content_layout.addWidget(self.pages)

        main_layout.addWidget(content_container)

    def _setup_pages(self):
        self.stats_widget = StatsWidget()
        self.stats_widget.stats_changed.connect(self._on_stats_modified)
        self.pages.addWidget(self.stats_widget)

        self.inventory_widget = InventoryWidget()
        self.inventory_widget.btn_add.clicked.connect(self._on_add_item)
        self.pages.addWidget(self.inventory_widget)

        self.world_widget = WorldWidget()
        self.world_widget.progress_changed.connect(self._on_progress_modified)
        self.pages.addWidget(self.world_widget)

        self.page_slots = QWidget()
        self.slots_layout = QVBoxLayout(self.page_slots)
        actions_layout = QHBoxLayout()
        self.btn_import = QPushButton("Import Character from another Save")
        self.btn_import.clicked.connect(self._import_character)
        self.btn_delete = QPushButton("Delete Selected Character")
        self.btn_delete.clicked.connect(self._delete_character)
        self.btn_delete.setStyleSheet("background-color: #a83232;")
        actions_layout.addWidget(self.btn_import)
        actions_layout.addWidget(self.btn_delete)
        self.slots_layout.addLayout(actions_layout)
        self.slots_layout.addWidget(QLabel("Select a character slot to edit:"))
        self.pages.addWidget(self.page_slots)

    def _load_styles(self):
        style_path = get_resource_path("src/ui/styles.qss")
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
            if item and item.widget() and isinstance(item.widget(), QPushButton):
                item.widget().setParent(None)
        
        if self.save_manager:
            for slot in self.save_manager.slots:
                status = "Active" if slot['active'] else "Empty"
                btn = QPushButton(f"Slot {slot['id']}: {slot['name']} ({status})")
                btn.clicked.connect(lambda checked=False, s_id=slot['id']: self._select_slot(s_id))
                self.slots_layout.addWidget(btn)

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
        self.lbl_status.setText(f"Editing Slot {slot_id}: {stats.name}")

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
                    QMessageBox.information(self, "Success", "Item added to inventory.")

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
