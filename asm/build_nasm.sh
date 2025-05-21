set -e
nasm -f elf64 -o test.o test.nasm
ld -s -o test test.o
./test
rm test.o
#rm test
