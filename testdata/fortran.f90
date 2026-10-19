! a free-form comment
#define WITH_MPI 1
module shapes
  implicit none
  private

  integer, parameter :: dp = selected_real_kind(15, 307)
  real(dp), parameter :: pi = 3.141592653589793_dp
  double precision :: legacy
  character(len=32) :: label = 'a ''quoted'' name'
  character(len=8) :: other = "double ""quoted"""
  logical :: flag = .TRUE.
  logical :: off = .false._1

  integer :: bits = B'1010'
  integer :: octal = O'755'
  integer :: hexed = Z'FF'
  real :: small = 1.5e-3
  real :: tiny_val = .5D0
  integer :: plain = -42
  integer :: kinded = 42_dp

  public :: area, perimeter

contains

  pure function area(r) result(a)
    real(dp), intent(in) :: r
    real(dp) :: a
    a = pi * r ** 2
  end function area

  subroutine perimeter(r, p)
    real(dp), intent(in) :: r
    real(dp), intent(out) :: p
    p = 2.0_dp * pi * r
  end subroutine perimeter

  subroutine report(n)
    integer, intent(in) :: n
    integer :: i
100 format (A, 1X, I5, '%', /)
    do i = 1, n
      if (i .EQ. 1 .OR. i .NE. n) then
        write (*, 100) 'step', i
      else if (i > n / 2) then
        go to 200
      end if
    end do
200 continue
    if (n < 0) error stop
  end subroutine report

end module shapes
