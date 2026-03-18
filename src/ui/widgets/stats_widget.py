from PySide6.QtWidgets import (
    QWidget,
    QFormLayout,
    QLineEdit,
    QSpinBox,
    QVBoxLayout,
    QGroupBox,
)
from PySide6.QtCore import Signal


class StatsWidget(QWidget):
    stats_changed = Signal()

    def __init__(self):
        super().__init__()
        self._setup_ui()

    def _setup_ui(self):
        layout = QVBoxLayout(self)

        # Identity Group
        group_identity = QGroupBox("Character Identity")
        form_identity = QFormLayout(group_identity)
        self.txt_name = QLineEdit()
        self.txt_name.textChanged.connect(self.stats_changed.emit)
        form_identity.addRow("Name:", self.txt_name)
        layout.addWidget(group_identity)

        # Stats Group
        group_stats = QGroupBox("Attributes")
        form_stats = QFormLayout(group_stats)

        self.spin_level = self._create_spin(1, 713, form_stats, "Level:")
        self.spin_souls = self._create_spin(0, 999999999, form_stats, "Souls:")

        self.spin_vigor = self._create_spin(1, 99, form_stats, "Vigor:")
        self.spin_mind = self._create_spin(1, 99, form_stats, "Mind:")
        self.spin_endurance = self._create_spin(1, 99, form_stats, "Endurance:")
        self.spin_strength = self._create_spin(1, 99, form_stats, "Strength:")
        self.spin_dexterity = self._create_spin(1, 99, form_stats, "Dexterity:")
        self.spin_intelligence = self._create_spin(1, 99, form_stats, "Intelligence:")
        self.spin_faith = self._create_spin(1, 99, form_stats, "Faith:")
        self.spin_arcane = self._create_spin(1, 99, form_stats, "Arcane:")

        layout.addWidget(group_stats)
        layout.addStretch()

    def _create_spin(self, min_val, max_val, layout, label):
        spin = QSpinBox()
        spin.setRange(min_val, max_val)
        spin.valueChanged.connect(self.stats_changed.emit)
        layout.addRow(label, spin)
        return spin

    def load_stats(self, stats_data):
        """Populates the UI with character stats."""
        self.txt_name.setText(stats_data.name)
        self.spin_level.setValue(stats_data.level)
        self.spin_souls.setValue(stats_data.souls)
        self.spin_vigor.setValue(stats_data.vigor)
        self.spin_mind.setValue(stats_data.mind)
        self.spin_endurance.setValue(stats_data.endurance)
        self.spin_strength.setValue(stats_data.strength)
        self.spin_dexterity.setValue(stats_data.dexterity)
        self.spin_intelligence.setValue(stats_data.intelligence)
        self.spin_faith.setValue(stats_data.faith)
        self.spin_arcane.setValue(stats_data.arcane)

    def get_stats(self):
        """Returns a dictionary of current UI values."""
        return {
            "name": self.txt_name.text(),
            "level": self.spin_level.value(),
            "souls": self.spin_souls.value(),
            "vigor": self.spin_vigor.value(),
            "mind": self.spin_mind.value(),
            "endurance": self.spin_endurance.value(),
            "strength": self.spin_strength.value(),
            "dexterity": self.spin_dexterity.value(),
            "intelligence": self.spin_intelligence.value(),
            "faith": self.spin_faith.value(),
            "arcane": self.spin_arcane.value(),
        }
