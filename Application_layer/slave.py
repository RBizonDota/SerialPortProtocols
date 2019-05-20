import sys
import subprocess
from PyQt5 import QtWidgets
import socket
import json
import threading
import time
conf = {
    "Name": "Hello my brother",
    "Baud": 11111,
    "FileName": "",
    "FileDir": ""
}


# class Settings (второе диалоговое окно)
class Second(QtWidgets.QWidget):
    def __init__(self, sock):
        super(Second, self).__init__()
        self.setWindowTitle('Settings')
        self.setStyleSheet(open("style.css", "r").read())
        self.resize(650, 270)
        self.sock = sock

    # функция для кнопки Settings, открытие 2 диалогового окна
    def settings_on_click(self):
        self.port_name = QtWidgets.QLineEdit(self)
        self.port_name.move(120, 20)
        self.port_name.setText(conf["Name"])
        self.port_name.resize(280, 30)
        self.label_port_name = QtWidgets.QLabel('Port Name', self)        
        self.label_port_name.move(20, 25)

        self.baud = QtWidgets.QLineEdit(self)
        self.baud.setText(str(conf["Baud"]))
        self.baud.move(120, 80)
        self.baud.resize(280, 30)
        self.label_baud = QtWidgets.QLabel('Baud', self)
        self.label_baud.move(20, 85)

        self.file_name = QtWidgets.QLineEdit(self)
        self.file_name.setText(conf["FileName"])
        self.file_name.move(120, 140)
        self.file_name.resize(280, 30)
        self.label_file_name = QtWidgets.QLabel('File Name', self)
        self.label_file_name.move(20, 145)


        self.choose_file = QtWidgets.QPushButton("Choose the file", self)
        self.choose_file.move(450, 130)
        self.choose_file.clicked.connect(self.file_on_click)

        self.submit = QtWidgets.QPushButton('Submit', self)
        self.submit.move(475, 200)
        self.submit.clicked.connect(self.submit_on_click)

        self.show()

    # функция для кнопки Submit во 2 диалоговом окне
    def submit_on_click(self):
        conf["Name"] = self.port_name.text()
        conf["Baud"] = int(self.baud.text())
        conf["FileName"] = self.file_name.text()
        self.type1()
        self.hide()

    def type1(self):
        msg = {
            "Type": 1,
            "Cnf": conf,
            "Data": ""
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.sendall(b"\n")
        
        
    # выбор файла
    def file_on_click(self):
        fname = QtWidgets.QFileDialog.getOpenFileName(self, 'Open file', '/home')[0]
        self.file_name.setText(str(fname))


# class главного окна
class MainWindow(QtWidgets.QWidget):
    def __init__(self):
        super(MainWindow, self).__init__()
        args = 'go run ../main.go ../common.go ../slave.go ../hamming.go ../socket.go'.split()
        self.proc = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,  # If not set - python stdin used
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        time.sleep(10)   
        self.setWindowTitle('Slave')
        self.setStyleSheet(open("style.css", "r").read())
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8889))
        t = threading.Thread(target=self.reader)
        t.start()
        self.UI()
        
    def reader(self):
        while True:
            data = self.sock.recv(1024)
            if not data:
                break
            parsed_data = json.loads(data)
            print("reader entry data = ",parsed_data)
            if parsed_data["Type"]==1:
                conf["Name"] = parsed_data["Cnf"]["Name"]
                conf["Baud"] = parsed_data["Cnf"]["Baud"]
                conf["FileName"] = parsed_data["Cnf"]["FileName"]
                conf["FileDir"] = parsed_data["Cnf"]["FileDir"]
                #print("conf modified")
            if parsed_data["Type"]==0:
                if parsed_data["Data"]=="Close":#Закрытие порта
                    self.onClose()
                if parsed_data["Data"]=="RST":#Разрыв соединения
                    self.onRST()
                if parsed_data["Data"]=="ConnectOK":#Успешное подключение
                    self.confirm_connection()
                if parsed_data["Data"]=="transmitRST":#Разрыв во время передачи TODO ПОКА НЕ ВПИСАНА В GOLANG
                    self.get_file_RST()
                if parsed_data["Data"]=="packetLoss":#Началась потеря пакетов
                    self.packet_loss()
    # задание сокета и передача данных
    #def read_json(self):
    #    data = self.sock.recv(1024)
    #    parsed_data = json.loads(data)
    #    if parsed_data['Type'] == 1:
    #        self.conf = parsed_data['Cnf']

    # задание вида главного диалогового окна
    def UI(self):
        self.open_port = QtWidgets.QPushButton("Open the port", self)
        self.settings = QtWidgets.QPushButton("Settings", self)
        self.open_port.move(180, 50)
        self.settings.move(200, 150)
        self.open_port.clicked.connect(self.port_on_click)
        self.w = Second(self.sock)
        self.settings.clicked.connect(self.w.settings_on_click)
    def closeEvent(self, event):
        self.sock.close()
        self.proc.terminate()
    # функция для кнопки Open port
    def port_on_click(self):
        if self.open_port.clicked:
            self.type0("OpenSlave")
            self.open_port.hide()
            self.close_port = QtWidgets.QPushButton("Close the port", self)
            self.close_port.move(180, 50)
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
        msg = {
            "Type": 0,
            "Cnf": conf,
            "Data": str
        }
        data = json.dumps(msg)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.sendall(b"\n")
        
    def packet_loss(self):
        pass
    def onRST(self):
        pass
    
    def get_file_RST(self):
        pass    

    def onClose(self):
        self.open_port.show()
        self.close_port.hide()

    def confirm_connection(self):
        pass
    
def main():
    app = QtWidgets.QApplication(sys.argv)
    w = MainWindow()
    w.show()
    sys.exit(app.exec_())


if __name__ == '__main__':
    main()
