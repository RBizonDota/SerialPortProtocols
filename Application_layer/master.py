# go run test.go socket.go common.go hamming.go

import sys
from PyQt5 import QtWidgets
import socket
import json
import pickle

conf = {
    "Name": "Hello my sister",
    "Baud": 41334,
    "FileName": "",
    "FileDir": ""
}

'''
# задание сокета и передача данных
def read_json():
    sock = socket.socket()
    sock.connect(('localhost', 8888))
    while True:
        data = sock.recv(1024)
        if not data:
            break
    parsed_data = json.loads(data)
    conf = parsed_data["Cnf"]
    sock.close()
'''

# class Settings (второе диалоговое окно)
class Second(QtWidgets.QWidget):
    def __init__(self):
        super(Second, self).__init__()
        self.setWindowTitle('Settings')

    # функция для кнопки Settings, открытие 2 диалогового окна
    def settings_on_click(self):
        self.port_name = QtWidgets.QLineEdit(self)
        self.port_name.setText(conf["Name"])
        self.port_name.move(120, 20)
        self.port_name.resize(280, 30)
        self.label_port_name = QtWidgets.QLabel('Port Name', self)
        self.label_port_name.move(20, 30)

        self.baud = QtWidgets.QLineEdit(self)
        self.baud.setText(str(conf["Baud"]))
        self.baud.move(120, 80)
        self.baud.resize(280, 30)
        self.label_baud = QtWidgets.QLabel('Baud', self)
        self.label_baud.move(20, 90)

        self.file_dir = QtWidgets.QLineEdit(self)
        self.file_dir.setText(conf["FileDir"])
        self.file_dir.move(120, 140)
        self.file_dir.resize(280, 30)
        self.label_file_dir = QtWidgets.QLabel('File Dir', self)
        self.label_file_dir.move(20, 150)

        self.choose_file = QtWidgets.QPushButton("Choose the dir", self)
        self.choose_file.move(400, 140)
        self.choose_file.clicked.connect(self.dir_on_click)

        self.submit = QtWidgets.QPushButton('Submit', self)
        self.submit.move(400, 200)
        self.submit.clicked.connect(self.submit_on_click)

        self.show()

    # функция для кнопки Submit во 2 диалоговом окне
    def submit_on_click(self):
        conf["Name"] = self.port_name.text()
        conf["Baud"] = int(self.baud.text())
        conf["FileDir"] = self.file_dir.text()
        self.type1()
        self.hide()

    # выбор папки
    def dir_on_click(self):
        dir = QtWidgets.QFileDialog.getExistingDirectory(self, 'Open file', '/home')
        self.file_dir.setText(str(dir))

    def type1(self):
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8888))
        msg = {
            "Type": 1,
            "Cnf": conf,
            "Data": ""
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.close()


# class главного окна
class MainWindow(QtWidgets.QWidget):

    def __init__(self):
        super(MainWindow, self).__init__()
        self.setWindowTitle('Master')
        self.resize(700, 200)
        self.UI()
        # read_json()

    # задание вида главного диалогового окна
    def UI(self):
        self.open_port = QtWidgets.QPushButton("Open the port", self)
        self.settings = QtWidgets.QPushButton("Settings", self)
        self.open_port.move(300, 50)
        self.settings.move(300, 125)
        self.open_port.clicked.connect(self.port_on_click)
        self.w = Second()
        self.settings.clicked.connect(self.w.settings_on_click)

    def type0(self, str):
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8888))
        msg = {
            "Type": 0,
            "Cnf": conf,
            "Data": str
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.close()

    # Open port button
    def port_on_click(self):
        if self.open_port.clicked:
            self.type0("Open")
            self.open_port.hide()
            self.close_port = QtWidgets.QPushButton("Close the port", self)
            self.open_connection = QtWidgets.QPushButton("Open the connection", self)
            self.close_port.move(100, 50)
            self.open_connection.move(400, 50)
            self.close_port.clicked.connect(self.close_port_click)
            self.open_connection.clicked.connect(self.open_connection_click)
            self.close_port.show()
            self.open_connection.show()

        else:
            self.error_dialog = QtWidgets.QErrorMessage()
            self.error_dialog.showMessage('Something went wrong!')

    # close the port button
    def close_port_click(self):
        self.type0("Close")
        self.open_port.show()
        if self.open_connection.isVisible():
            self.close_port.hide()
            self.open_connection.hide()
        else:
            self.close_connection.hide()
            self.get_file.hide()
            self.close_port.hide()

    # open the connection button
    def open_connection_click(self):
        if self.open_connection.clicked:
            self.type0("ConnInit")
            self.open_connection.hide()
            self.close_connection = QtWidgets.QPushButton("Close the connection", self)
            self.get_file = QtWidgets.QPushButton("Get the file", self)
            self.resume = QtWidgets.QPushButton("Resume", self)
            self.close_connection.move(280, 50)
            self.get_file.move(500, 50)
            self.resume.move(500, 125)
            self.close_port.clicked.connect(self.close_port_click)
            self.close_connection.clicked.connect(self.close_connection_click)
            self.get_file.clicked.connect(self.get_file_click)
            self.resume.clicked.connect(self.resume_click)
            self.close_connection.show()
            self.get_file.show()
            self.resume.show()
        else:
            self.error_dialog = QtWidgets.QErrorMessage()
            self.error_dialog.showMessage('Something went wrong!')

    # resume button
    def resume_click(self):
        self.type0("transmitResume")

    # get the file button
    def get_file_click(self):
        self.type0("transmitInit")

    # close the connection button
    def close_connection_click(self):
        self.type0("ConnEnd")
        self.open_port.show()
        self.close_port.hide()
        self.close_connection.hide()
        self.get_file.hide()

def main():
    app = QtWidgets.QApplication(sys.argv)
    w = MainWindow()
    w.show()
    sys.exit(app.exec_())


if __name__ == '__main__':
    main()
