import sys
from PyQt5 import QtWidgets
import socket
import json

conf = {
    "Name": "Hello my brother",
    "Baud": 11111,
    "FileName": "",
    "FileDir": ""
}


# class Settings (второе диалоговое окно)
class Second(QtWidgets.QWidget):
    def __init__(self):
        super(Second, self).__init__()
        self.setWindowTitle('Settings')

    # функция для кнопки Settings, открытие 2 диалогового окна
    def settings_on_click(self):
        self.port_name = QtWidgets.QLineEdit(self)
        self.port_name.move(120, 20)
        self.port_name.setText(conf["Name"])
        self.port_name.resize(280, 30)
        self.label_port_name = QtWidgets.QLabel('Port Name', self)
        self.label_port_name.move(20, 30)

        self.baud = QtWidgets.QLineEdit(self)
        self.baud.setText(str(conf["Baud"]))
        self.baud.move(120, 80)
        self.baud.resize(280, 30)
        self.label_baud = QtWidgets.QLabel('Baud', self)
        self.label_baud.move(20, 90)

        self.file_name = QtWidgets.QLineEdit(self)
        self.file_name.setText(conf["FileName"])
        self.file_name.move(120, 140)
        self.file_name.resize(280, 30)
        self.label_file_name = QtWidgets.QLabel('File Name', self)
        self.label_file_name.move(20, 150)

        self.choose_file = QtWidgets.QPushButton("Choose the file", self)
        self.choose_file.move(400, 140)
        self.choose_file.clicked.connect(self.file_on_click)

        self.submit = QtWidgets.QPushButton('Submit', self)
        self.submit.move(400, 200)
        self.submit.clicked.connect(self.submit_on_click)

        self.show()

    # функция для кнопки Submit во 2 диалоговом окне
    def submit_on_click(self):
        conf["Name"] = self.port_name.text()
        conf["Baud"] = self.baud.text()
        conf["FileName"] = self.choose_file.text()
        self.type1()
        self.hide()

    def type1(self):
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8889))
        msg = {
            "Type": 1,
            "Cnf": conf,
            "Data": ""
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.close()

    # выбор файла
    def file_on_click(self):
        fname = QtWidgets.QFileDialog.getOpenFileName(self, 'Open file', '/home')[0]
        self.file_name.setText(str(fname))


# class главного окна
class MainWindow(QtWidgets.QWidget):
    def __init__(self):
        super(MainWindow, self).__init__()
        self.setWindowTitle('Slave')
        self.UI()

    # задание сокета и передача данных
    def read_json(self):
        data = self.sock.recv(1024)
        parsed_data = json.loads(data)
        if parsed_data['Type'] == 1:
            self.conf = parsed_data['Cnf']

    # задание вида главного диалогового окна
    def UI(self):
        self.open_port = QtWidgets.QPushButton("Open the port", self)
        self.settings = QtWidgets.QPushButton("Settings", self)
        self.open_port.move(200, 50)
        self.settings.move(200, 100)
        self.open_port.clicked.connect(self.port_on_click)
        self.w = Second()
        self.settings.clicked.connect(self.w.settings_on_click)

    # функция для кнопки Open port
    def port_on_click(self):
        if self.open_port.clicked:
            self.type0("Open")
            self.open_port.hide()
            self.close_port = QtWidgets.QPushButton("Close the port", self)
            self.close_port.move(200, 50)
            self.close_port.clicked.connect(self.close_port_click)
            self.close_port.show()
        else:
            self.error_dialog = QtWidgets.QErrorMessage()
            self.error_dialog.showMessage('Something went wrong!')

    # close the port button
    def close_port_click(self):
        self.type0("Close")
        self.open_port.show()
        self.close_port.hide()

    def type0(self, str):
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8889))
        msg = {
            "Type": 0,
            "Cnf": conf,
            "Data": str
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.close()

def main():
    app = QtWidgets.QApplication(sys.argv)
    w = MainWindow()
    w.show()
    sys.exit(app.exec_())


if __name__ == '__main__':
    main()
