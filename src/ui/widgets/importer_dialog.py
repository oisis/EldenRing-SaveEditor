from PySide6.QtWidgets import (QDialog, QVBoxLayout, QListWidget, 
                             QPushButton, QLabel, QFileDialog, QMessageBox)
from core.save_manager import SaveManager

class ImporterDialog(QDialog):
    def __init__(self, parent=None):
        super().__init__(parent)
        self.setWindowTitle("Import Character")
        self.setMinimumSize(400, 500)
        self.source_manager = None
        self.selected_slot_id = None
        
        self._setup_ui()

    def _setup_ui(self):
        layout = QVBoxLayout(self)
        
        self.lbl_info = QLabel("Select source save file:")
        layout.addWidget(self.lbl_info)
        
        self.btn_browse = QPushButton("Browse Source Save")
        self.btn_browse.clicked.connect(self._browse_source)
        layout.addWidget(self.btn_browse)
        
        self.list_slots = QListWidget()
        layout.addWidget(self.list_slots)
        
        self.btn_import = QPushButton("Import Selected Slot")
        self.btn_import.setEnabled(False)
        self.btn_import.clicked.connect(self.accept)
        layout.addWidget(self.btn_import)

    def _browse_source(self):
        file_path, _ = QFileDialog.getOpenFileName(
            self, "Open Source Elden Ring Save", "", "Save Files (*.sl2 *.txt);;All Files (*)"
        )
        if file_path:
            try:
                self.source_manager = SaveManager(file_path)
                self.source_manager.load()
                self.list_slots.clear()
                for slot in self.source_manager.slots:
                    status = "Active" if slot['active'] else "Empty"
                    self.list_slots.addItem(f"Slot {slot['id']}: {slot['name']} ({status})")
                self.btn_import.setEnabled(True)
                self.lbl_info.setText(f"Source: {file_path}")
            except Exception as e:
                QMessageBox.critical(self, "Error", f"Failed to load source save: {str(e)}")

    def get_selected_slot_id(self):
        return self.list_slots.currentRow()
