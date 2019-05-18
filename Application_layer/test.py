
import subprocess
import threading
import time

def async_log(proc):
	try:
		while(True):
			data = proc.stdout.readline()
			print(data.decode("utf-8"))
	except:
		print("---Err in worker")

def async_reader(proc):
    while(True):
        data = proc.stderr.readline()
        print(data.decode('utf-8'))#Парсинг tcpMessage!!!!!!
        if (data =="SetCNF"):
            data2 = proc.stderr.readline()
            print("CNF = "+data2.decode('utf-8'))

def connInit(cmdstring):
    print('Python process started!')
    # May be shell command or something else ...
    args = cmdstring.split()

    # subprocess want slice of args, like: ['ping', '-c1', 'mail.ru']
    proc = subprocess.Popen(
        args,
        stdin=subprocess.PIPE,  # If not set - python stdin used
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return proc


connection = connInit("go run ../main.go ../hamming.go ../common.go ../master.go")
data = bytes("Open|", 'utf-8')
print(connection.stdin.write(data))
time.sleep(20)
data = bytes("OUT|", 'utf-8')
print(connection.stdin.write(data))
