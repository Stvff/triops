nasm -f elf64 -o test.o test.nasm
ld -o test test.o
./test
rm test.o
#rm test
