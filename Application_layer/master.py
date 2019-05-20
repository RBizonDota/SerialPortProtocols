# go run test.go socket.go common.go hamming.go

import sys
import threading
import subprocess
from PyQt5 import QtWidgets
import socket
import json
import time

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
    def __init__(self, sock):
        super(Second, self).__init__()
        self.setWindowTitle('Settings')
        self.setStyleSheet(open("style.css", "r").read())
        self.resize(650, 270)
        self.sock = sock
        
    # функция для кнопки Settings, открытие 2 диалогового окна
    def settings_on_click(self):
        self.port_name = QtWidgets.QLineEdit(self)
        self.port_name.setText(conf["Name"])
        self.port_name.move(120, 20)
        self.port_name.resize(280, 30)
        self.label_port_name = QtWidgets.QLabel('Port Name', self)
        self.label_port_name.move(20, 25)

        self.baud = QtWidgets.QLineEdit(self)
        self.baud.setText(str(conf["Baud"]))
        self.baud.move(120, 80)
        self.baud.resize(280, 30)
        self.label_baud = QtWidgets.QLabel('Baud', self)
        self.label_baud.move(20, 85)

        self.file_dir = QtWidgets.QLineEdit(self)
        self.file_dir.setText(conf["FileDir"])
        self.file_dir.move(120, 140)
        self.file_dir.resize(280, 30)
        self.label_file_dir = QtWidgets.QLabel('File Dir', self)
        self.label_file_dir.move(20, 145)

        self.choose_file = QtWidgets.QPushButton("Choose the dir", self)
        self.choose_file.move(450, 130)
        self.choose_file.clicked.connect(self.dir_on_click)

        self.submit = QtWidgets.QPushButton('Submit', self)
        self.submit.move(475, 200)
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
        self.file_dir.setText(str(dir)+"/")

    def type1(self):
        msg = {
            "Type": 1,
            "Cnf": conf,
            "Data": ""
        }
        data = json.dumps(msg)
        print(data)
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.sendall(b"\n")
        

# class главного окна
class MainWindow(QtWidgets.QWidget):

    def __init__(self):
        super(MainWindow, self).__init__()
        self.setWindowTitle('Master')
        self.resize(900, 350)
        args = 'go run ../main.go ../common.go ../master.go ../hamming.go ../socket.go'.split()
        self.proc = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,  # If not set - python stdin used
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        ) 
        #self.sock = socket.socket()
        #self.sock.connect(('localhost', 8888))
        # read_json()
        time.sleep(10)
        self.setStyleSheet(open("style.css", "r").read())
        self.sock = socket.socket()
        self.sock.connect(('localhost', 8888))
        t = threading.Thread(target=self.reader)
        t.start()
        self.UI()
        
    def closeEvent(self, event):
        self.sock.close()
        self.proc.terminate()
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
                conf["ResumeCounter"]=parsed_data["Cnf"]["ResumeCounter"]
                if parsed_data["Data"]=="U":
                    self.update_status(0)
                if parsed_data["Data"]=="RU":
                    self.update_status(self.resume_num)
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
                if parsed_data["Data"]=="TRERROR":
                    self.transmit_error()
                if parsed_data["Data"]=="packetLoss":#Началась потеря пакетов
                    self.packet_loss()
                if parsed_data["Data"]=="transmitOK":
                    self.confirm_transmit()

    def transmit_error(self):
        self.status.setText("Error while transmiting")
        self.close_port.setEnabled(True)
        self.close_connection.setEnabled(True)
        self.resume.setEnabled(True)
        self.get_file.setEnabled(True)
        self.settings.setEnabled(True)


    def update_status(self, min_num):
        curData = conf["ResumeCounter"]["CurName"]+conf["ResumeCounter"]["CurFile"]
        maxData = conf["ResumeCounter"]["NameCadrSize"]+conf["ResumeCounter"]["FileCadrSize"]
        if curData<min_num:
            curData = min_num
        self.status.setText("Transmiting Data transmited "+str(curData)+"/"+str(maxData)+" ("+str(int(curData/maxData*100))+"%)")
         
    # задание вида главного диалогового окна
    def UI(self):
        self.open_port = QtWidgets.QPushButton("Open the port", self)
        self.settings = QtWidgets.QPushButton("Settings", self)
        self.status = QtWidgets.QLabel("Initial state", self)
        self.status.setText("Initial state")
        self.status.resize(400,30)
        self.status.move(300, 25)
        self.status.show()
        self.open_port.move(385, 100)
        self.settings.move(400, 200)
        self.open_port.clicked.connect(self.port_on_click)
        self.w = Second(self.sock)
        self.settings.clicked.connect(self.w.settings_on_click)
        self.close_connection = QtWidgets.QPushButton("Close the connection", self)
        self.close_connection.hide()
        self.get_file = QtWidgets.QPushButton("Get the file", self)
        self.get_file.hide()
        self.resume = QtWidgets.QPushButton("Resume", self)
        self.resume.hide()
        

    def type0(self, str):
        msg = {
            "Type": 0,
            "Cnf": conf,
            "Data": str
        }
        data = json.dumps(msg)
        print(data)
        #self.sock.send(bytes(data, 'utf-8'))
        self.sock.sendall(bytes(data, 'utf-8'))
        self.sock.sendall(b"\n")
        #self.sock.close()

    # Open port button
    def port_on_click(self):
        if self.open_port.clicked:
            self.type0("Open")
            self.status.setText("Port opened")
            self.open_port.hide()
            self.close_port = QtWidgets.QPushButton("Close the port", self)
            self.open_connection = QtWidgets.QPushButton("Open the connection", self)
            self.close_port.move(200, 100)
            self.open_connection.move(500, 100)
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
        self.onClose()
        self.resume.hide()

    def onClose(self):
        self.open_port.show()
        if self.open_connection.isVisible():
            self.close_port.hide()
            self.open_connection.hide()
        else:
            self.close_connection.hide()
            self.get_file.hide()
            self.close_port.hide()

    def confirm_transmit(self):
        self.status.setText("File successfully transmited")
        self.close_port.setEnabled(True)
        self.close_connection.setEnabled(True)
        self.resume.setEnabled(True)
        self.get_file.setEnabled(True)
        self.settings.setEnabled(True)

    # open the connection button
    def open_connection_click(self):
        if self.open_connection.clicked:
            self.type0("ConnInit")
            self.status.setText("Connecting...")
            self.open_connection.setEnabled(False)
            self.settings.setEnabled(False)
            self.close_port.setEnabled(False)
        else:
            self.error_dialog = QtWidgets.QErrorMessage()
            self.error_dialog.showMessage('Something went wrong!')

    def confirm_connection(self):
        self.open_connection.hide()
        self.close_connection.move(400, 100)
        self.get_file.move(650, 100)
        self.resume.move(650, 200)
        self.close_connection.clicked.connect(self.close_connection_click)
        self.get_file.clicked.connect(self.get_file_click)
        self.resume.clicked.connect(self.resume_click)
        self.close_connection.show()
        self.get_file.show()
        self.resume.show()
        self.open_connection.setEnabled(True)
        self.settings.setEnabled(True)
        self.close_port.setEnabled(True)
        self.status.setText("Connected")
    # resume button
    def resume_click(self):
        self.resume_num = conf["ResumeCounter"]["CurName"]+conf["ResumeCounter"]["CurFile"]
        self.type0("transmitResume")

    # get the file button
    def get_file_click(self):
        self.type0("transmitInit")
        self.close_port.setEnabled(False)
        self.close_connection.setEnabled(False)
        self.resume.setEnabled(False)
        self.get_file.setEnabled(False)
        self.settings.setEnabled(False)

    def get_file_RST(self):
        pass
    # close the connection button
    def close_connection_click(self):
        self.type0("ConnEnd")
        self.onRST()

    def onRST(self):
        self.status.setText("Connection broken")
        #self.open_connection.hide()
        #self.open_port.show()
        #self.close_port.hide()
        self.close_connection.hide()
        self.open_connection.setDisabled(False)
        self.open_connection.show()
        self.get_file.hide()    
        self.settings.setDisabled(False)
        self.close_port.setDisabled(False)
        self.resume.hide()
        

    def packet_loss(self):
        pass
    

def main():
    app = QtWidgets.QApplication(sys.argv)
    w = MainWindow()
    w.show()
    sys.exit(app.exec_())


if __name__ == '__main__':
    main()
